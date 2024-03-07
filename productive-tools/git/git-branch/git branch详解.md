
---
git branch 详解
---

# 1 branch分支简介
branch(分支)是git中最核心的概念，代码都是在某一个特定的分支上开发的。基于不同的需求，一般我们会创建不同的分支。
常见的git flow工作流是下面这样的:
dev 分支 频繁变化的一个分支
test 分支 供测试与产品等人员使用的一个分支，变化不是特别频繁
master分支，生产环境发布分支，变化非常不频繁的一个分支
feature分支，开发人员开发具体功能的分支，一般基于dev分支创建而来
bugfix分支，生产系统中出现了紧急bug, 用于紧急修复的分支。

当我们基于某个分支(譬如master)创建一个新分支时(譬如dev)，此时这两个分支其实内容都是一模一样的，然后随着我们对两个分支
代码的修改，它们会开始分叉， 指针会指向各自最新的提交点，类似下面这样：

![git-branch.png](images%2Fgit-branch.png)

新提交的parent指针会指向上一次提交，类似一个链表，通过parent指针将分支上的所有提交记录连接起来，方便进行版本回退。

# 2 基本操作命令

```shell
# 查看有哪些分支
git branch -a
# 查看有哪些分支，并展示这些分支的最新提交
git branch -av
# 切换到另一个分支
git checkout dev 或 git switch dev
# 创建一个新的分支
git branch test
# 基于当前分支创建一个新分支并切换到这个新分支上
git checkout -b bugfix
# 删除一个分支, 如果该分支有提交未进行合并，则会删除失败
git branch -d test
# 强制删除一个分支
git branch -D test

# 注意：删除分支前需要切换到另一个分支上，我们是无法在当前分支上删除当前分支的。

# 删除一个远程分支(删除远程dev分支)
git push origin --delete dev
# 查看所有远程分支
git remote show
# 查看远程分支URL
git remote show origin
# 将远程dev分支的代码合并到本地dev分支(假设此时处于本地dev分支上)
git merge origin dev
# 查看某次提交的修改内容
git show commit_id

# 我想从别人正在工作的远程分支拉出(checkout)一个分支
# 首先, 从远程拉取(fetch) 所有分支:
# git fetch --all
# 假设你想要从远程的bug_fix分支拉出到本地的bug_fix
# git checkout --track origin/bug_fix
# (--track 是 git checkout -b [branch] [remotename]/[branch] 的简写)
# 这样就得到了一个daves分支的本地拷贝, 任何推过(pushed)的更新，远程都能看到.

# 基于远程仓库创建新分支并且切换到新分支
git checkout -b branch_name remote_name/branch
git checkout -b test origin/test

# 将远程代码回滚到某个版本
# 首先将远程分支对应的本地分支代码回滚到指定版本
git reset --hard HEAD~1
# 2、加入-f参数，强制提交，远程端将强制更新到reset版本
git push -f

# git commit规范,详细说明此次commit的功能
git commit -m <type>: <subject>
# type:
# feature: 新功能（feature）
# fix: 修补bug、style等
# refactor: 重构（即不是新增功能，也不是修改bug的代码变动）
# test: 增加测试 chore: 构建过程或辅助工具的变动

# subject
# 提交目的的简短描述，描述做了啥或者改了啥，如果有团队管理工具(issue ,JIRA)或者
# 产品需求，必须以内部命名的需求代号作为描述信息的一部分，方便查看日志，合并和cherry-pick。

# 举例：
git commit -m "feature:开发完成#代号 XXX.XXX需求" 
git commit -m "fix:修改 #代号 XXXX查询问题" 
```

# 3 在github上创建代码仓库

![new-repository.png](images%2Fnew-repository.png)

>配置代码仓库的公私钥
```shell
which ssh-keygen
/usr/bin/ssh-keygen
# 生成公私钥
ssh-keygen
# 然后一路回车即可
```
![config-ssh.png](images%2Fconfig-ssh.png)

从中可以看到生成的公私钥文件存放的位置，将公钥的内容拷贝，而后在github的代码仓库中进行公钥配置。

```shell
cd /Users/qiujun/.ssh/
cat id_rsa.pub
```

![config-public-key.png](images%2Fconfig-public-key.png)

# 4 git pull和git push命令详解
远程仓库是为了解决多人分布式协作开发项目而产生的，当某个程序员在dev分支开发了一段代码后，其他的开发
人员想要看到这段代码，怎么办？只需要这个程序员将修改通过git push指令推送到远程dev分支，其他人通过
git pull 命令拉取远程dev分支的最新变更到本地dev分支即可。
> git pull = git fetch + git merge
> git push - u origin master 等同于
> git branch  --set-upstream-to=origin/master + git push origin master
前者是将远程仓库的origin的master分支与本地仓库的master分支相关联。
后者是将本地的master分支推送到远程。

## 4.1 代码冲突和解决
当两个及以上的coder都对同一个文件的同一行代码进行了修改了，便会产生代码冲突，此时需要我们手动的
去解决冲突。一般而言，都是后推送的人会遇到冲突，解决冲突需要你跟冲突方进行协商，到底采用谁的修改。

t1: 开发人员a在dev分支上对test.txt文件进行了修改，增加了第三行"How old are you?", 并进行了
commit提交，但是还未推送到远程。

![git-conflict-scene-one.png](images%2Fgit-conflict-scene-one.png)

t2: 此时开发人员b也在dev分支上对test.txt文件进行了修改，增加了第三行"You should go home 
before it is too late"。

![git-conflict-scene-two.png](images%2Fgit-conflict-scene-two.png)

t3: 此时开发人员a将变更推送到远程，由于开发人员b还未推送到远程，所以执行是没有问题的。

![git-conflict-scene-three.png](images%2Fgit-conflict-scene-three.png)

t4: 但之后开发人员b将自己的修改推送到远程时，便会产生冲突了。因为两个人修改的是同一行，git无法进行自动合并。

git会首先提醒你推送失败，因为远程仓库包含你本地尚不存在的提交，你需要将远程仓库的更新拉取到本地。
当你执行git pull时，实际上就是先git fetch，然后git merge，将远程dev分支的变更合并到本地的dev分支，
此时冲突便产生了。
git提示你自动合并失败，出现冲突的文件是test.txt

![git-conflict-scene-four.png](images%2Fgit-conflict-scene-four.png)

t5: 现在你需要手动修改test.txt文件，解决冲突。

![git-conflict-scene-five.png](images%2Fgit-conflict-scene-five.png)

很明显，下面的"How old are you"是其他人的修改，而上面的"You should go home before it is too late"是你
自己的修改，通过commit id也能分辨出来，此时你需要决定采用谁的修改。假设通过与a的协商，决定采用你的修改，那么你
就删除最后三行，保留自己的修改。解决冲突后，git add，git commit 提交，(解决冲突后会产生一次新的提交，此时你会
领先远程dev分支两个提交) 最后推送到远程，此时便没问题了。

![git-conflict-scene-six.png](images%2Fgit-conflict-scene-six.png)

t6:此时开发人员a去拉取远程dev分支的代码，第三行便变成了你的修改"You should go home before it is too late"
![git-conflict-scene-seven.png](images%2Fgit-conflict-scene-seven.png)

当你在本地分支修改并提交后，推送到远程前，使用git status命令会发现自己领先远程分支一个提交

![git-status.png](images%2Fgit-status.png)

# 5 本地分支和远程分支
当你新建一个本地分支(譬如test)并进行代码修改后，想要推送到远程时，git会提示你对应远程分支不存在，此时你需要执行
git push --set-upstream origin test命令，新建远程origin test分支并追踪本地test分支。

![git-local-link-remote.png](images%2Fgit-local-link-remote.png)

也可以使用更简单的命令 git push -u origin test，效果是一样的。
![git-push-u-remote-branch.png](images%2Fgit-push-u-remote-branch.png)

如果本地分支不存在，也可以通过以下命令创建一个新的本地分支并与远程分支相关联
```shell
git checkout -b test origin/test
```
```shell
# git push 的完整写法是
git pull origin src:dest
# git push --set-upstream origin test 的完整写法是
git push --set-upstream origin local:remote
# Branch local set up to track remote branch remote from origin
```

## 5.1 如何删除远程分支？

有两种方法可以删除远程分支，一是: 推送一个空分支到对应的远程分支，即可将该远程分支删除
```shell
git push origin :dest
```

第二种方式是直接删除远程分支：
```shell
git push --delete origin test
```
![git-delete-remote-branch.png](images%2Fgit-delete-remote-branch.png)


>将本地的标签推送到远程以及删除远程的标签

```shell
# 推送本地的一个或多个标签到远程
git push origin v1.0 v2.0
# 推送本地的一个或多个标签到远程的完整写法
git push origin refs/tags/v1.0:refs/tags/v1.0
# 将本地的所有tags都推送到远程
git push origin --tags
# 删除远程的tag
git push origin :refs/tags/v1.0
# 或者也可以使用下面的命令
git push origin --delete tag v2.0
# 删除本地标签
git tag -d v3.0 
# 展示远程master分支的提交历史记录
git log origin/master
git log remotes/origin/master
```

![git-look-up-tags.png](images%2Fgit-look-up-tags.png)

![git-tag-operation.png](images%2Fgit-tag-operation.png)

# 6 git rebase
变基，意思是改变分支的根基。
从某种程度上来说，rebase与merge可以完成类似的工作，不过二者的工作方式有着显著的差异。
merge合并两个分支是分叉的，而rebase是一条平直的线。 

rebase注意事项:
rebase过程中也会出现冲突
解决冲突后，使用git add添加，然后执行
```shell
git rebase --continue
```
接下来git会继续应用余下的补丁
任何时候都可以通过如下命令终止rebase, 分支会恢复到rebase开始前的状态
```shell
git rebase --abort
```

**rebase最佳实践**
不要对master分支执行rebase, 否则会引起很多问题。
一般来说，执行rebase的分支都是自己的本地分支，没有推送到远程版本库。

```shell
# 我想撤销rebase/merge
# 你可以合并(merge)或rebase了一个错误的分支, 或者完成不了一个进行中的rebase/merge。 
# git在进行危险操作的时候会把原始的HEAD保存在一个叫ORIG_HEAD的变量里, 所以要把分支恢复
# 到rebase/merge前的状态是很容易的。
git reset --hard ORIG_HEAD
# 有时候这些合并非常复杂，你应该使用可视化的差异编辑器(visual diff editor):
git mergetool -t opendiff
```

# 7 git cherry-pick
对于多分支的代码库，将代码从一个分支转移到另一个分支是常见需求。
这时分两种情况。一种情况是，你需要另一个分支的所有代码变动，那么就采用合并（ git merge）。
另一种情况是，你只需要部分代码变动（某几个提交），这时可以采用 cherry-pick。

## 7.1 基本用法
git cherry-pick命令的作用，就是将指定的提交（commit）应用于其他分支。

```shell
git cherry-pick <commitHash>
```
上面命令就会将指定的提交commitHash，应用于当前分支。这会在当前分支产生一个新的提交，当然它们的哈希值会不一样。
举例来说，代码仓库有master和feature两个分支。

```shell
a - b - c - d   Master
         \
           e - f - g Feature
```

现在将提交f应用到master分支。
```shell
# 切换到master分支
git checkout master
# Cherry pick 操作
git cherry-pick f
```
上面的操作完成以后，代码库就变成了下面的样子。
```shell
a - b - c - d - f   Master
         \
           e - f - g Feature
```

从上面可以看到，master分支的末尾增加了一个提交f。
git cherry-pick命令的参数，不一定是提交的哈希值，分支名也是可以的，表示转移该分支的最新提交。

```shell
git cherry-pick feature
```
上面代码表示将feature分支的最近一次提交，转移到当前分支。

## 7.2 转移多个提交
cherry-pick 支持一次转移多个提交。
```shell
git cherry-pick <HashA> <HashB>
```

上面的命令将 A 和 B 两个提交应用到当前分支。这会在当前分支生成两个对应的新提交。
如果想要转移一系列的连续提交，可以使用下面的简便语法。

```shell
git cherry-pick A..B 
```
上面的命令可以转移从 A 到 B 的所有提交(左开右闭区间)。它们必须按照正确的顺序放置：
提交 A 必须早于提交 B，否则命令将失败，但不会报错。注意，使用上面的命令，提交 A 将
不会包含在 cherry-pick 中。如果要包含提交 A，可以使用下面的语法。

```shell
git cherry-pick A^..B 
```

## 7.3 配置项
git cherry-pick命令的常用配置项如下。
（1）-e，--edit
打开外部编辑器，编辑提交信息。
（2）-n，--no-commit
只更新工作区和暂存区，不产生新的提交。
（3）-x
在提交信息的末尾追加一行(cherry picked from commit ...)，方便以后查到这个提交是如何产生的。
（4）-s，--signoff
在提交信息的末尾追加一行操作者的签名，表示是谁进行了这个操作。
（5）-m parent-number，--mainline parent-number
如果原始提交是一个合并节点，来自于两个分支的合并，那么cherry-pick 默认将失败，因为它不知道应该
采用哪个分支的代码变动。
-m配置项告诉git，应该采用哪个分支的变动。它的参数parent-number是一个从1开始的整数，代表原始提交
的父分支编号。

```shell
git cherry-pick -m 1 <commitHash>
```
上面命令表示，cherry-pick 采用提交commitHash来自编号1的父分支的变动。
一般来说，1号父分支是接受变动的分支（the branch being merged into），2号父分支是作为变动来源的分支
（the branch being merged from）。

## 7.4 解决cherry-pick过程中的代码冲突
如果操作过程中发生代码冲突，cherry-pick 会停下来，让用户决定如何继续操作。
（1）--continue
用户解决代码冲突后，第一步将修改的文件重新加入暂存区（git add .），第二步使用下面的命令，
让cherry-pick过程继续执行。

```shell
git cherry-pick --continue
```
2）--abort
发生代码冲突后，放弃合并，回到操作前的样子。
```shell
git cherry-pick --abort
```
（3）--quit
发生代码冲突后，退出 cherry-pick，但是不回到操作前的样子。
```shell
git cherry-pick --quit
```

## 7.5 转移至另一个代码仓库
cherry-pick 也支持转移另一个代码库的提交，方法是先将该库加为远程仓库。

```shell
git remote add target git://gitUrl
```
上面命令添加了一个远程仓库target。 然后将远程代码拉取到本地。

```shell
git fetch target
```
上面命令将远程代码仓库拉取到本地。
接着，检查一下要从远程仓库转移的提交，获取它的哈希值。

```shell
git log target/master
```
最后，使用git cherry-pick命令转移提交。

```shell
git cherry-pick <commitHash>
```

# 8 合并多个commit
在使用 git 作为版本控制的时候，我们可能会由于各种各样的原因提交了许多临时的 commit，而这些
commit 拼接起来才是完整的任务。那么我们为了避免太多的 commit 而造成版本控制的混乱，通常我们
推荐将这些 commit 合并成一个。

![git-log.png](images%2Fgit-log.png)

譬如，我们需要将 2dfbc7e8 和 c4e858b5 合并成一个 commit，那么我们可以输入如下命令:

```shell
git rebase -i f1f92b
# 或者
git rebase -i HEAD~2
```

其中，-i 的参数是不需要合并的 commit 的 hash 值，这里指的是第一条 commit， 接着我们就进入到
vi 的编辑模式:

![git-rebase-edit.png](images%2Fgit-rebase-edit.png)

可以看到其中分为两个部分，上方未注释的部分是填写要执行的指令，而下方注释的部分则是指令的提示说明。
指令部分中由前方的命令名称、commit hash 和 commit message 组成。

当前我们只要知道 pick 和 squash 这两个命令即可。
-  pick 的意思是要会执行这个 commit
-  squash 的意思是这个 commit 会被合并到前一个commit

我们将 c4e858b5 这个 commit 前方的命令改成 squash 或 s，然后输入:wq以保存并退出

![git-rebase-edit-detail.png](images%2Fgit-rebase-edit-detail.png)

此时我们会看到 commit message 的编辑界面：

![git-rebase-commit-edit.png](images%2Fgit-rebase-commit-edit.png)

其中, 非注释部分就是两次的 commit message, 你要做的就是将这两个修改成新的 commit message。

![git-rebase-commit-done.png](images%2Fgit-rebase-commit-done.png)

输入wq保存并推出, 再次输入git log查看 commit 历史信息，你会发现这两个commit已经合并了。

![git-rebase-result.png](images%2Fgit-rebase-result.png)

注意事项：如果这个过程中有操作错误，可以使用 git rebase --abort来撤销修改，回到没有开始操作合并之前的状态。

如果你想组合这些提交(commit) 并重命名这个提交(commit), 你应该在第二个提交(commit)旁边添加一个r，或者更简单
的用s 替代 f:

```shell
pick a9c8a1d Some refactoring
pick 01b2fd8 New awesome feature
s b729ad5 fixup
s e3851e8 another fix
```