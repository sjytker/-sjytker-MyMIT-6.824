



## json.NewEncoder 无法写入文件

**解决**：

​	struct 可导出性的问题，必须 uppercase 开头的变量才是可导出的，才能被写入 json 文件





## rpc 调用找不到 master 的函数

**问题**：

​	我在 master 写了 2 个 rpc 调用函数，其中一个能正常调用。另一个报错找不到该方法。调用方式完全一致，并且 master 也确定有 worker 调用的同名函数。

​	（搞了半天才找到问题）

**解决**：

​	还是 Golang exported 类型的问题。无论是 type，还是参数，都只有 uppercase 开头才是 exported，否则 unexported。

​	在 unexported 的情况下，文件不能写入，rpc 也无法调用。





## Mutex 的问题

**问题***：

​	习惯了 Java 的可重入锁，没意识到 golang 的 mutex 是不可重入的



## 其他逻辑错误





