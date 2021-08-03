# 简介
一个通过命令行操作redis服务器的cmd工具  
![image](https://user-images.githubusercontent.com/47658310/128036299-5f08252e-ec71-4bd5-8a89-811ab02acd49.png)



# 为什么要写这个工具
1、受不了某某Manager加载数据库列表慢  
2、受不了某某Manager大数量量情况下查询key值慢  
3、受不了某某Manager不支持忽略key的大小写查询  
4、更受不了某某Manager操作中程序崩溃退出  
9、主要是还收费  

# 优势
1、操作简单快捷  
2、纯命令行操作，无图形界面的渲染  
3、使用多线程同时加载多个数据库信息，极大的提升数据库列表信息加载效率  
4、支持不区分大小写缓存key查询方式  
5、支持模糊查询的缓存key来删除缓存  
6、免费、有问题就及时处理、更新、优化  
7、支持环境切换

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
