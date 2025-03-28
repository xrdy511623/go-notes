
---
git alias 详解
---
git 命令，尤其是比较长的命令使用起来不太方便，拼写也容易出错，为了提高工作效率，我们可以
使用git命令的别名来提高效率。

```shell
git config --global alias.st status
git config --global alias.ch checkout
git config --global alias.cm commit
git config --global alias.ad add
git config --global alias.br branch
git config --global alias.unstage 'reset HEAD'
git config --global alias.cp cherry-pick
```
可以通过~/.gitconfig文件查看配置是否成功

![check-git-alias.png](images%2Fcheck-git-alias.png)

之后，我们便可以使用alias来操作使用git了
```shell
# 查看当前分支状态
git st
# 切换到test分支
git ch test
# 提交代码
git cm -m "feature:xxx"
# 将xxa到xxb之间的提交(包含xxb)转移提交到当前分支
git cp xxa..xxb
```