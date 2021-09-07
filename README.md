# 简介
一个通过命令行操作redis服务器的cmd工具  
![image](https://user-images.githubusercontent.com/47658310/132305237-b2bccc4e-b63b-4a19-ab2e-424073b344e5.png)



# 为什么要写这个工具
1、受不了某某Manager加载数据库列表慢  
2、受不了某某Manager大数量量情况下查询key值慢  
3、受不了某某Manager不支持忽略key的大小写查询  
4、更受不了某某Manager操作中程序崩溃退出  
9、主要是还收费  

# 优势
1、支持不区分大小写的批量查询、删除操作
2、纯命令行操作，配置系统环境变量后可以随时轻量级唤醒程序
3、支持多线程、多连接配置，根据数据量调整连接数提升操作性能
4、支持多环境配置文件实时切换
5、免费、有问题就及时处理、更新、优化  

# 使用文档
## 下载源文件自己编译打包
1、你需要clone这个项目到本地（当然你也可以直接复制代码到本地，反正代码也不多）  
2、你需要有一套完整的go语言开发环境  
3、你要熟悉go程序的打包方式  

## windows可执行程序
[点击下载](https://raw.githubusercontent.com/pwzos/rediscmd/main/target/rediscmd.exe)

## macos/linux可执行程序
[点击下载](https://github.com/pwzos/rediscmd/raw/main/target/rediscmd)

## 配置环境变量
对应系统配置好环境变量，可以直接在命令行中输入rediscmd即可唤醒此工具
