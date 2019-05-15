# puff
puff是一个基于vasc框架的代码自动生成工具。它可以生成函数映射、导入配置文件、并静态化程序运行时需要依赖的资源。
### 使用方法：
语法：
```
puff -h
Usage of puff:
  -c string
        vasc config file
  -i string
        input source file directory
  -o string
        output source file
```
-i 指定用于导入go源代码文件的目录。puff会递归扫描此路径下所有子目录下的go源文件。此参数一般设置为项目的controller路径。  
-o 指定生成的程序代码路径。  
-c 指定vasc配置文件路径。vasc配置文件的格式请参考vasc工程。

### 说明：
puff生成的go代码文件格式如下：  
```
function map
  函数映射。由-i参数指定的代码路径下源代码中的所有导出函数。是一个字符串到函数的map映射。
vasc configuration
  vasc配置文件的内容。puff会解析由-c参数指定的配置文件的内容，并将文件内容放置在此处。
application configuration
  应用层配置文件的内容。应用层配置文件包括web路由配置、定时任务和队列任务的配置信息。
main函数
  这里包括对vasc框架服务初始化和信号处理的封装。
```
