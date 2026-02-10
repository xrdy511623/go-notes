
---
git基础命令详解
---

# 1 git 基础配置和获取命令帮助

## 1.1 git配置文件的存储位置
/etc/gitconfig文件: 对系统所有用户生效，如果你传递参数--system给git config，它将明确的读和写这个文件
~/.gitconfig文件，仅对当前用户生效，如果你传递参数--global给git config，它将读写此文件。
位于代码仓库git目录下的config文件，也就是.git/config，无论当前用户是谁，都采用此配置文件的设置。每个级别重写前一个级别的值。
因此，在.git/config中的值覆盖了在/etc/gitconfig中的同一个值。

## 1.2 配置用户名和密码
当安装Git后首先要做的事情是设置用户名和邮箱l地址。这是非常重要的，因为每次Git提交都会使用该信息。它被永远的嵌入到了你的提交中：

```shell
git model --global user.name "张三"
git model --global user.email "zangsan@sina.com"
```

要针对特定代码仓库生效，则使用--local, 对系统所有用户生效，则使用--system
```shell
git model --local user.name "john adams"
git model --local user.email "ybzdqhl@gmail.com"
```


## 1.3 配置比较工具
另外一个你可能需要配置的有用的选项是缺省的比较工具，它用来解决合并时的冲突。例如，想使用 vimdiff 作为比较工具。
```shell
git model --global merge.tool vimdiff
```


## 1.4 检查配置
如果想检查你的设置，可以使用 git config --list 命令来列出Git可以在该处找到的所有的设置:
```shell
git model --list
```

## 1.5 添加删除配置项

添加配置项，使用:
如果不指定，默认是添加到local配置中。
```shell
git model [--local|--global|--system] --add key value
```

删除配置项，使用:
```shell
git model [--local|--global|--system] --unset key
```


## 1.6 获取帮助
```shell
git help model
git model --help
man git-model
```

# 2 基础命令说明

## 2.1  git init, git add,  git commit, git status

```shell
# git init 命令初始化一个git仓库, 此时的默认分支是master分支
git init
# 使用以下命令可以修改分支名
git branch -m name

# git status 查看工作区的状态, 当我们修改或新建文件时，执行此命令提示我们工作区存在
# 尚未跟踪的文件，需要使用git add 建立跟踪。

# add命令表示将改动提交到暂存区，便于git跟踪文件变更

# 表示将文件filename的改动提交到暂存区
git add filename
# 表示将所有文件的改动提交到暂存区
git add *
# 表示所有文件的改动提交到暂存区, 但是在.gitignore过滤范围内的除外。
git add .

# 当我们执行完git add后，文件的修改便从工作区被提交到了暂存区，此时我们需要执行git commit命令
# 将文件改动提交，以纳入git的分布式版本控制系统中, 此时工作区的状态就是clean的了。
git commit -m "annotation"

# 查看指定提交具体的修改情况
git show commit_id
# 也可以将add和commit两个命令合并
git commit -am "initial commit"

# 如果commit的备注信息写错了，但是还没有push推送到远程，可以使用以下命令进行修改
git commit --amend --only
# 这会打开你的默认编辑器, 在这里你可以编辑信息. 另一方面, 你也可以用一条命令一次完成:
git commit --amend --only -m 'xxxxxxx'

# 修改提交的用户名和邮箱
git commit --amend --author "New Authorname <authoremail@mydomain.com>"

# 从一个提交里移除一个文件
git checkout HEAD^ myfile
git add -A
git commit --amend

# 我意外的做了一次硬重置(hard reset)，我想找回我的内容
# 如果你意外的做了 git reset --hard, 你通常能找回你的提交(commit), 因为Git对每件事都会有日志，且都会保存几天。
git reflog
# 你将会看到一个你过去提交(commit)的列表, 和一个重置的提交。 选择你想要回到的提交(commit)的SHA，再重置一次:
git reset --hard commit_id

# 我想把未暂存的内容移动到另一个已存在的分支new_branch
git stash
git checkout new_branch
git stash pop

# 我想丢弃某些未暂存的内容
# 可以stash 你不需要的部分, 然后stash drop。
git stash -p
git stash drop

```



## 2.2 git reset

```shell
# 如果想要撤销commit提交，回退到指定某一次提交的版本，可以使用reset命令。
git reset [--soft|--mixed|--hard] HEAD^/commit_id
git reset --hard HEAD^^
# 撤销最近的四次提交
git reset --hard HEAD~4
# 重置某个特殊的文件, 你可以用文件名做为参数:
git reset filename
# 重置分支到你所需的提交
git reset --hard c5bc55a

# 我想扔掉本地的提交(commit)，以便我的分支与远程的保持一致，
# 先确认你没有推(push)你的内容到远程。
# git status 会显示你领先(ahead)源(origin)多少个提交:
git status
# 然后丢弃本地的提交，与远程分支保持一致
git reset --hard origin/my-branch
# 错误删除了一个分支，想要恢复这个分支
git checkout -b my-branch
# 通过git reflog查看所有提交日志，使用reset重置到你需要的提交
git reflogs
git reset --hard 4e3cd85


# -mixed 为默认，可以不用带该参数，用于重置暂存区的文件与上一次的提交(commit)保持一致，工作区文件内容保持不变。
# 也就是说不删除工作空间改动代码，撤销commit，并且撤销git add操作
# git reset --mixed HEAD^ 和 git reset HEAD^ 效果是一样的。
# –soft 参数用于回退到某个版本，不删除工作空间改动代码，撤销commit，不撤销git add
# –hard 参数撤销工作区中所有未提交的修改内容，将暂存区与工作区都回到上一次版本，并删除之前的所有信息提交。也就是既
# 删除工作空间改动代码，也撤销commit，还撤销 git add


# HEAD 说明：
#- HEAD 表示当前版本
#- HEAD^ 上一个版本
#- HEAD^^ 上上一个版本
#- HEAD^^^ 上上上一个版本
#  可以使用 ～数字表示
#- HEAD~0 表示当前版本
#- HEAD~1 上一个版本
#- HEAD~2 上上一个版本
#- HEAD~3 上上上一个版本
#
# 也可以通过指定commit id回退到指定版本。
# 首先，我们通过git log 命令查看当前分支的提交历史记录，找到我们想要回退的版本。
# 然后执行git reset命令进行回退


#如果想要撤销add操作，也就是取消暂存区的改动，可使用以下命令：
git restore --staged filename

#如果是想要取消当前工作区的修改，可使用以下命令:
git restore filename
```



## 2.3 git stash

```shell
# git stash的作用是将目前已经修改但是不想commit提交的内容暂存到堆栈中，后续可以在某个分支上将暂存的修改内容恢复出来。
# 也就是说，stash中的内容不仅仅可以恢复到原先开发的分支，也可以恢复到其他任意指定的分支上。git stash作用的范围包括工作
# 区和暂存区中的内容，也就是说没有提交的内容都会保存至堆栈中。能够将所有未提交的修改（工作区和暂存区）保存至堆栈中，用于后续恢复。
git stash
# 作用等同于git stash， 区别是可以加一些注释
git stash save "temporary save"
# 查看当前stash中的内容
git stash list
# 将当前stash中的内容弹出，并应用到当前分支对应的工作目录上。
# 注：该命令将堆栈中最近保存的内容删除（栈是先进后出）
git stash pop
# 如果从stash中恢复的内容和当前目录中的内容发生了冲突，也就是说，恢复的内容和当前目录修改了
# 同一行的数据，那么会提示报错，需要解决冲突，可以通过创建新的分支来解决冲突。

# 将堆栈中的内容应用到当前目录，不同于git stash pop，该命令不会将内容从堆栈中删除，
# 也就说该命令能够将堆栈的内容多次应用到工作目录中，适应于多个分支的情况。
git stash apply

# 从堆栈中移除某个指定的stash
git stash drop name

# 清除堆栈中的所有内容
git stash clear

# 查看堆栈中最新保存的stash和当前目录的差异。
git stash show

# 从最新的stash创建分支。
git stash branch
# 应用场景：当储藏了部分工作，暂时不去理会，继续在当前分支进行开发，后续想将stash中的内容
# 恢复到当前工作目录时，如果是针对同一个文件的修改（即便不是同行数据），那么可能会发生冲突，
# 恢复失败，这里通过创建新的分支来解决。可以用于解决stash中的内容和当前目录的内容发生冲突
# 的情景。发生冲突时，需手动解决冲突。
# 然后你可以apply某个stash
git stash apply "stash@{n}"
# 此处， 'n'是stash在栈中的位置，最上层的stash会是0
```


## 2.4 git tag, git blame和git diff**
```shell
# 新建标签，标签有两种: 轻量级标签(lightweight)与带有附注标签(annotated)
# 创建一个轻量级标签
git tag v1.0.1
# 创建一个带有附注的标签
git tag -a v1.0.1 -m "v1.0.1 released version"
# 删除标签
git tag -d tag_name
# 查看标签
git tag
# 查看标签
git tag  -l  'v1.0'
git tag  -l  '*2'

# 查看代码的历史修改记录
git blame filename
# 查看文件n1行到n2行修改记录
git blame -L  n1, n2 filename

# 比较的是暂存区与工作区文件之间的差别
git diff 
# 比较的是最新的提交与工作区之间的差别
git diff HEAD 
# 比较的是最新的提交与暂存区之间的差别
git diff -cached 
```


## 2.5 gitignore

```.gitignore
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool, specifically when used with LiteIDE
*.out

# Dependency directories (remove the comment below to include it)
# vendor/
```


## 2.6 跟踪文件, 比较和git tag**

```shell
# 我只想改变一个文件名字的大小写，而不修改内容
git mv --force myfile MyFile
# 我想从Git删除一个文件，但保留该文件
git rm --cached log.txt
# 比较本地分支与远程分支的不同
git diff [本地分支名] origin/[远程分支名]
创建tag 【tag名】
git tag v1.0
# 查看存在的tag
git tag
# 将tag更新到远程
git push origin --tags

# 删除标签
git tag -D tag_name

# 显示暂存区和工作区的差异
git diff

# 显示暂存区和上一个commit的差异【文件名】
git diff --cached [hell.txt]

# 显示工作区与当前分支最新commit之间的差异
git diff HEAD

# 显示两次提交之间的差异【分支名】
git diff [first-branch]...[second-branch]
```