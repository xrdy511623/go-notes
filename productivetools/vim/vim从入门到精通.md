
---
vim 从入门到精通
---

**如何在120分钟内掌握vim?**
 
> 一个前提: 键盘映射 (基本上不会对你现在的键盘使用习惯做更改，但是能提高80%的效率)



![mac-switch-key.png](images%2Fmac-switch-key.png)



![windows-switch-key.png](images%2Fwindows-switch-key.png)



> 一个定则: 二八定则 (用20%的时间去学我们常用的80%的功能)
> 一个要求: 不用鼠标 (鼠标是影响你速度的重要原因，我们现在将不使用鼠标完成所有文本操作)
> 一个方法: 可以联系 (熟能生巧，刻意练习是掌握技巧的不二法门)
> 一个重要的事情: 得不到的重复技能会遗忘，得不到的重复技能会遗忘，得不到的重复技能会遗忘，重要的事情说三遍
> 一个终极目标: 写作速度与思维速度一致


![basic.png](images%2Fbasic.png)



![install.png](images%2Finstall.png)



![run-and-quit.png](images%2Frun-and-quit.png)



**三种模式简介**
普通模式: 按下Esc键可切换到普通模式
插入模式: 按i或a或c可切换到插入模式(编辑模式)
可视模式: 按v进入普通可视模式，按V进入行可视模式，control/command+v进入块可视模式



![advanced.png](images%2Fadvanced.png)



**基于单词的移动**

命令                      光标动作
w                正向移动到下一单词的开头
b                反向移动到当前单词/上一单词的开头
e                正向移动到当前单词/下一单词的结尾
ge               反向移动到上一单词的结尾


>对字符进行查找和通过查找进行移动

命令               用途
f{char}         正向移动到下一个{char}所在之处
F{char}         反向移动到上一个{char}所在之处
t{char}         正向移动到下一个{char}所在之处的前一个字符上
T{char}         反向移动到上一个{char}所在之处的后一个字符上
;               重复上次的字符查找命令
,               反转方向查找上次的字符查找命令



![selection-zone.png](images%2Fselection-zone.png)



>分隔符文本对象

a表示around是包含符号(括号、引号等)在内；
而i表示in是表示符号(括号、引号等)内的内容，不包括符号本身。

命令                 选择区域
a)或ab              一对圆括号
i)或ib              圆括号内部
a}或aB              一对花括号
i}或iB              花括号内部
a]                  一对方括号
i]                  方括号内部
a>                  一对尖括号
i>                  尖括号内部
a'                  一对单引号
i'                  单引号内部
a"                  一对双引号
i"                  双引号内部
a`                  一对反引号
i`                  反引号内部
at                  一对XML标签
it                  XML标签内部


> 范围文本对象

文本对象             选择范围
iw                  当前单词
aw                  当前单词及一个空格
iW                  当前字串
aW                  当前字串及一个空格
is                  当前句子
as                  当前句子及一个空格
ip                  当前段落
iw                  当前段落及一个空行


> 操作符待决模式{motion}
{motion}指的就是: 分隔符文本对象和范围文本对象

d{motion}     删除模式         dd删除一行
c{motion}     修改模式         cc修改一行
y{motion}     复制模式         yy复制一行
v{motion}     可视模式           -


> 设置标记，快速回跳

m{mark}      设置标记
`{mark}      返回标记


> 复制与粘贴

y  复制
p  粘贴

> 查找与替换

/{pattern}                      查找     使用n跳转
/%s/{pattern}/{string}/g        替换     使用c进行替换确认


> 翻页

control+f      下翻一页
control+b      上翻一页
control+d      下翻半页
control+u      上翻半页



![keyboard.png](images%2Fkeyboard.png)



安装sublime及其配置
下载安装: https://www.sublimetext.com/3
Package Control安装: https://packagecontrol.io/installation

如何安装sublime的插件？
在安装好sublime及其Package Control后，按control/command + shift + p，输入Install在下拉菜单选择
Install Package按下回车键，在接着弹出的输入框中输入插件名字，回车选择即可安装该插件。

>常用的插件

ConvertToUTF8           支持多种编码，解决中文乱码问题
Bracket Highlighter     用于高亮匹配括号、引号、html标签
DocBlockr               可以自动生成PHPDoc风格的注释
Emmet                   快速编写HTML，原Zen Coding。
SideBar Enhancements    这个插件改进了侧边栏，增加了许多功能
evernote                这个支持markdown语法
markdown preview        markdown预览插件， alt+m开启


> 如何启动Vim模式

在菜单栏中: Preferences->Setting->User，即可打开配置文件进行编辑，将ignored_packages项目里的[]里面内容清空



![open-vim-pattern.png](images%2Fopen-vim-pattern.png)