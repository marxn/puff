package main

import "flag"
import "fmt"
import "bytes"
import "os"
import "os/exec"
import "go/ast"
import "go/parser"
import "go/token"
import "io/ioutil"
import "reflect"
import "strings"
import "encoding/json"
import "github.com/marxn/vasc/global"

type funcItem struct {
    Comment  []string
    FuncName   string
}

type directoryInfo struct {
    Dir           string
    FileList    []string
    FuncList    []funcItem
    //CommentList []string
    NeedExport    bool
}

func emptyDir(path string) bool {
    cmd := fmt.Sprintf("cd ..;find %s -name '*.go'", path)
    ret, err := ExecShellCmd(cmd)
    if err!=nil {
        return true
    }
    if ret == "" {
        return true
    }
    return false
}

func getDirList(path string) ([]string, error) {
    cmd := fmt.Sprintf("cd ..;find %s -type d", path)
    ret, err := ExecShellCmd(cmd)
    if err!=nil {
        fmt.Println(err)
        return nil, err
    } 
    
    var result []string
    for _, value := range strings.Split(ret, "\n") {
        if value!="" && value != "." && value != ".." && !emptyDir(value) {
            result = append(result, value)
        }
    }
    
    return result, nil
}

func getFileList(path string) ([]string, error) {
    cmd := fmt.Sprintf("cd ..;find %s -maxdepth 1 -name '*.go'", path)
    ret, err := ExecShellCmd(cmd)
    if err!=nil {
        return nil, err
    } 
    
    var result []string
    for _, value := range strings.Split(ret, "\n") {
        if value!="" {
            result = append(result, "../" + value)
        }
    }
    
    return result, nil
}

func loadVascConfigFile(fileName string) (*global.VascConfig, error) {
    config, err := ioutil.ReadFile(fileName)
    if err != nil{
        return nil, err
    }
    
    jsonResult := new(global.VascConfig)
    err = json.Unmarshal([]byte(config), jsonResult)
    if err != nil {
        return nil, err
    }
    
    return jsonResult, err
}

func isExported(funcName string) bool {
    nameBytes := []byte(funcName)
    if len(nameBytes) > 0 {
        return nameBytes[0] >= 65 && nameBytes[0] <= 90
    }
    return false
}

func getExportFuncList(fileList []string) ([]funcItem, error) {
    var result []funcItem
    for _, filename := range fileList {
        fset := token.NewFileSet()
        f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
        if err != nil {
            panic(err)
        }
        for _, value := range f.Decls {
            t := reflect.TypeOf(value)
            s := t.String()
            if strings.Contains(s, "FuncDecl") {
                decl := value.(*ast.FuncDecl)
                funcName := decl.Name
                if !isExported(fmt.Sprintf("%s", funcName)) {
                    continue
                }
                funcDesc := &funcItem{FuncName: fmt.Sprintf("%s", funcName)}
                doc  := decl.Doc
                if doc!=nil {
                    for _, docItem := range doc.List {
                        headByte := []byte(docItem.Text)
                        if string(headByte[0:3])=="///" {
                            if funcDesc.Comment==nil {
                                funcDesc.Comment = make([]string, 0)
                            }
                            funcDesc.Comment = append(funcDesc.Comment, string(headByte[3:]))
                        }
                    }
                }
                if funcDesc.Comment!=nil {
                    result = append(result, *funcDesc)
                }
            }
        }
    }
    return result, nil
}

func qualifyPath(path string) string {
    seg := strings.Split(path, "/") 
    if len(seg)==1 {
        return path
    }
    return seg[len(seg) - 1]
}

func ExecShellCmd(s string) (string, error) {
    cmd := exec.Command("/bin/bash", "-c", s)
    var out bytes.Buffer
    cmd.Stdout = &out
    err := cmd.Run()
    return out.String(), err
}

func replacePackagePath(path string) string {
    return strings.Replace(path, "/", "_", -1 )
}

func main() {
    var handlerHolder  []string
    var scheduleHolder []string
    var taskHolder     []string
    var groupHolder    []string
    var initHolder       string
    
    input, _           := os.Getwd()
    output             := input
    
    vascConfigFileName := flag.String("c", "", "vasc config file")
    
    flag.Parse()

    if *vascConfigFileName == "" {
        fmt.Println("invalid arguments")
        return
    }
    
    projectName, err := ExecShellCmd("basename " + input)
    if err!=nil {
        panic(err)
    }
    
    projectName = strings.TrimSpace(projectName)
    
    sourceInfo := make(map[string]*directoryInfo)
    
    dirList, err := getDirList(projectName)
    if err!=nil {
        panic(err)
    }
    
    for _, value := range dirList {
        fmt.Println("analyzing directory:" + value)
        dirInfo := new(directoryInfo)
        dirInfo.Dir = value
        fileList, err := getFileList(value)
        if err!=nil {
            panic(err)
        }
        
        dirInfo.FileList = fileList
        funcList, err := getExportFuncList(fileList)
        if err!=nil {
            panic(err)
        }
        if len(funcList) > 0 {
            dirInfo.FuncList   = funcList
            dirInfo.NeedExport = true
        }
        sourceInfo[value] = dirInfo
    }
    
    source := fmt.Sprintf("//Vasc generated code. Do not modify.\n\npackage main\n\nimport \"github.com/marxn/vasc\"\nimport \"github.com/marxn/vasc/global\"\n")
    
    for _, value := range sourceInfo {
        if value.NeedExport {
            source += fmt.Sprintf("import %s \"%s\"\n", replacePackagePath(value.Dir), value.Dir)
        }
    }
    
    source += "\n\nvar VascFuncMap = map[string]interface{}{\n"
    var exportTask []string
    
    for _, sourceCode := range sourceInfo {
        for _, funcCall := range sourceCode.FuncList {
            packagePrefix := replacePackagePath(sourceCode.Dir)
            source += fmt.Sprintf("    \"%s:%s\": %s.%s,\n", packagePrefix, funcCall.FuncName, packagePrefix, funcCall.FuncName)
            if funcCall.Comment!=nil {
                for _, comments := range funcCall.Comment {
                    defination := []byte(comments)
                    funcName := fmt.Sprintf("%s:%s", packagePrefix, funcCall.FuncName)
                    if string(defination[0:7])=="HANDLER" {
                        handlerHolder = append(handlerHolder, string(defination[7:]) + fmt.Sprintf(", \"route_handler\": \"%s\"", funcName))
                    } else if string(defination[0:10])=="MIDDLEWARE" {
                        groupHolder = append(groupHolder, string(defination[10:]) + fmt.Sprintf(", \"middleware\": \"%s\"", funcName))
                    } else if string(defination[0:8])=="SCHEDULE" {
                        scheduleHolder = append(scheduleHolder, string(defination[8:]) + fmt.Sprintf(", \"handler\": \"%s\", \"schedule_key\": \"%s\"", funcName, funcName))
                    } else if string(defination[0:4])=="TASK" {
                        taskHolder = append(taskHolder, string(defination[4:]) + fmt.Sprintf(", \"handler\": \"%s\", \"task_key\":\"%s\"", funcName, funcName))
                        exportTask = append(exportTask, fmt.Sprintf("const %s = \"%s\"", funcCall.FuncName, funcName))
                    } else if string(defination[0:4])=="INIT" {
                        initHolder = fmt.Sprintf("%s.%s", packagePrefix, funcCall.FuncName)
                    }
                }
            }
        }
    }
    source += fmt.Sprintf("}\n")
    
    configFile, err := ioutil.ReadFile(*vascConfigFileName)
    if err!=nil {
        panic(err)
    }
    source += fmt.Sprintf("\n\nvar configFile = `%s`", configFile)
    
    appConfigFile := fmt.Sprintf("\n{\n")
    appConfigFile += fmt.Sprintf("        \"schedule_list\": [\n")
    for index, schedule := range scheduleHolder {
    appConfigFile += fmt.Sprintf("            {%s}", schedule)
    if index < len(scheduleHolder) - 1 {
        appConfigFile += fmt.Sprintf(",")
    }
    appConfigFile += fmt.Sprintf("\n")
    }
    appConfigFile += fmt.Sprintf("        ],\n")
    
    appConfigFile += fmt.Sprintf("        \"task_list\": [\n")
    for index, task := range taskHolder {
    appConfigFile += fmt.Sprintf("            {%s}", task)
    if index < len(taskHolder) - 1 {
        appConfigFile += fmt.Sprintf(",")
    }
    appConfigFile += fmt.Sprintf("\n")
    }
    appConfigFile += fmt.Sprintf("        ],\n")

    appConfigFile += fmt.Sprintf("        \"webserver_route\": [\n")
    for index, handler := range handlerHolder {
    appConfigFile += fmt.Sprintf("            {%s}", handler)
    if index < len(handlerHolder) - 1 {
        appConfigFile += fmt.Sprintf(",")
    }
    appConfigFile += fmt.Sprintf("\n")
    }
    appConfigFile += fmt.Sprintf("        ],\n")
    appConfigFile += fmt.Sprintf("        \"webserver_route_group\": [\n")
    for index, group := range groupHolder {
    appConfigFile += fmt.Sprintf("            {%s}", group)
    if index < len(groupHolder) - 1 {
        appConfigFile += fmt.Sprintf(",")
    }
    appConfigFile += fmt.Sprintf("\n")
    }
    appConfigFile += fmt.Sprintf("        ]\n")
    appConfigFile += fmt.Sprintf("}\n")
    
    source += fmt.Sprintf("\n\nvar appConfigFile = `%s`\n", appConfigFile)

    source += fmt.Sprintf("func main() {\n")
    source += fmt.Sprintf("    err := vasc.InitInstance(\n")
    source += fmt.Sprintf("        &global.VascApplication{\n")
    source += fmt.Sprintf("            FuncMap: VascFuncMap,\n")
    source += fmt.Sprintf("            Configuration: configFile,\n")
    source += fmt.Sprintf("            AppConfiguration: appConfigFile,\n")
    source += fmt.Sprintf("    })\n\n")
    
    source += fmt.Sprintf("    if err!=nil {\n")
    source += fmt.Sprintf("        panic(err)\n")
    source += fmt.Sprintf("        return\n")
    source += fmt.Sprintf("    }\n")
    source += fmt.Sprintf("    defer vasc.Close()\n")
    source += fmt.Sprintf("\n")
    if initHolder != "" {
        source += fmt.Sprintf("    vasc.SetInitializer(%s)\n", initHolder)
    }
    source += fmt.Sprintf("    err = vasc.StartService()\n")
    source += fmt.Sprintf("    if err!=nil {\n")
    source +=             "        vasc.ErrorLog(\"Starting service failed: %s\", err.Error())\n"
    source += fmt.Sprintf("        return\n")
    source += fmt.Sprintf("    } else {\n")
    source += fmt.Sprintf("        vasc.Wait()\n")
    source += fmt.Sprintf("    }\n")
    source += fmt.Sprintf("}\n")

    err = ioutil.WriteFile(output + "/puff_main.go", []byte(source), 0666)
    if err != nil {
        fmt.Println("Cannot write output file:" + err.Error())
        os.Exit(-1)
    }
    /* 
    if len(exportTask) > 0 {
        sdkSource := fmt.Sprintf("package task\n\n")
        
        for _, exportTaskItem := range exportTask {
            sdkSource += fmt.Sprintf("%s\n", exportTaskItem)
        }
        
        sdkPath := fmt.Sprintf("%s/task", output)
        err := os.MkdirAll(sdkPath,os.ModePerm)
        if err != nil {
            fmt.Println("Cannot make task directory: " + err.Error())
            os.Exit(-1)
        }
        
        err = ioutil.WriteFile(sdkPath + "/PuffConstant.go", []byte(sdkSource), 0666)
        if err != nil {
            fmt.Println("Cannot create constant file: " + err.Error())
            os.Exit(-1)
        }
    }
    */
    fmt.Printf("%s finished.\n", output)
}
