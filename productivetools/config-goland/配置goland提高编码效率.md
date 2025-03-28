
---
配置好goland，提高编码效率
---

作为一个gopher，我日常编码主要是使用goland这款IDE，下面分享下我配置goland以提高编码效率的经验。

# 1 定制快捷键

## 1.1 快速打开终端

日常编码时，时不时会在本地调试代码，跑个脚本啥的，这时候需要打开终端，如果按照常规的方式选中目录，右键选择Open In
再选择Terminal那就太慢了，这里推荐设置一下快捷键(我设置为T，代表terminal终端)。
Preferences-->Keymap-->Plugins




![config-terminal.png](images%2Fconfig-terminal.png)





![config-terminal-with-shortcut.png](images%2Fconfig-terminal-with-shortcut.png)




## 1.2 快速创建go文件

Preferences-->Keymap-->Main Menu-->File-->File Open Actions-->New-->Go File
点击右键选择添加Add Keyboard Shortcut(快捷键)。我个人设置为G，代表go文件




![new-go-file.png](images%2Fnew-go-file.png)




## 1.3 快速创建普通文件

同样的，Preferences-->Keymap-->Main Menu-->File-->File Open Actions-->New-->File
点击右键选择添加Add Keyboard Shortcut(快捷键)。我个人设置为F，代表普通文件。




![new-normal-file.png](images%2Fnew-normal-file.png)




## 1.4 快速创建目录

同样的，Preferences-->Keymap-->Main Menu-->File-->File Open Actions-->New-->Create new directory or package
点击右键选择添加Add Keyboard Shortcut(快捷键)。我个人设置为D，代表目录。




![new-dir.png](images%2Fnew-dir.png)




## 1.5 左右切文件对比

有时候需要比较代码或者配置文件异同，或是参照，此时可以选中你要比较或参照的文件，右键选择Split Right，即可将其切到右侧




![split-right.png](images%2Fsplit-right.png)




![comparasion.png](images%2Fcomparasion.png)




# 2 安装好用的插件

https://www.tabnine.com/blog/top-goland-ide-plugins/

## 2.1 如何在goland中安装插件？

通过集成市场安装插件是一件简单的事情。
导航到 GoLand | Preferences | Plugins(如果你使用的是 macOS)
导航到插件并使用搜索栏找到你要安装的插件




![navigate-plugins.png](images%2Fnavigate-plugins.png)



只需单击安装即可，GoLand 将负责其余的工作。你可能需要重新启动 IDE 才能使插件生效。不过别担心，Goland 会提示你这样做。

你还可以从文件安装插件。在同一窗口中，单击齿轮图标，然后单击从磁盘安装插件。




![install-plugin-from-disk.png](images%2Finstall-plugin-from-disk.png)




## 2.2 Top 10插件

### 2.2.1 字符串操作(String Manipulation)

你是否曾经需要在编写代码时操作一些文本并为必须手动执行许多操作而感到遗憾?String Manipulation 字符串操作通过一长串可以修改文
本字符串的方法来解决这个问题。你可以随机排列文本行、更改大小写以及添加或删除转义字符。


### 2.2.2 Tabnine AI 代码补全

良好的代码补全可以为你节省大量键入代码的时间。代码补全是大多数 IDE 的共同特征，但并非所有代码补全都是一样的。Tabnine 使用 AI 
根据上下文预测你接下来可能想要输入的内容，而不是简单地根据你已经编写的内容列出所有可能的选项。



### 2.2.3  GitToolBox

如果你使用 Git，我觉得你应该使用 Git，这个插件将添加一些功能，让你的生活更轻松。显示 inline blame、提交编号和日期是 GitToolBox 
最有价值的功能之一。




![gittoolbox-demo.png](images%2Fgittoolbox-demo.png)




### 2.2.4 Protocol Buffers

Protocol Buffers[8] 是 Google 对轻量级序列化数据结构的实现。它的工作方式与 XML 类似，并且支持多种语言，包括 Go。如果你打算使用 
Protocol Buffers，此扩展将提供你需要的支持。


### 2.2.5 Key Promoter X

当你熟悉一个新的 IDE 时，你并不知道所有的快捷方式。有时你甚至会查找它们，但很快就忘记了，因为你使用它们的次数还不够多。
Key Prompter X 通过在你每次使用鼠标菜单时发送弹出通知来帮助你熟悉键盘快捷键，方便你记忆，将来使用键盘快捷键。


### 2.2.6 Makefile Language

Makefile 支持是必不可少的，尤其是在使用大型 makefile 时。这个插件提供了自动完成、语法高亮和一个 make 工具窗口——你在
IDE 中处理 Makefile 所需的一切。



### 2.2.7 .ignore

一个方便的 .ignore 文件生成器和编辑器。如果你正在使用 Git，可能需要忽略一些被 checked 的文件。此插件允许你从 GoLand 
中编辑忽略文件。


### 2.2.8 CSV

CSV 是常用的文件扩展名。这不是一个很好的文件扩展名，但有时你以 CSV 格式获取数据，需要对其进行处理。CSV 插件可让你做到这一点。


### 2.2.9 Rainbow Brackets

嵌套括号可能是噩梦，尤其是当它们聚集在一起或相距很远时。Rainbow brackets 为每对括号提供了不同的颜色，这样可以更容易地一目了然
地看到每个括号是否关闭，以及你当前处于哪个级别。


### 2.2.10 Gopher

这是一个进度条。它超级可爱。如果你喜欢所有可爱的东西，这可能是此列表中最重要的插件。