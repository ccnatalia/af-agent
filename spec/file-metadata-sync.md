# 文件 MD5 Manifest 规范

## 1. 目标

使用一个文本文件保存每个文件的路径和内容 MD5，用它记录最后一次成功同步的文件状态。

本方案不使用哈希树、路径哈希、文件大小、修改时间、权限或其他文件系统属性。

## 2. Manifest 文件

默认文件名：

```text
file-md5.txt
```

每行保存一条文件记录，格式为：

```text
MD5,path
```

示例：

```text
9E107D9D372BB6826BD81D3542A419D6,assets/app.tar.gz
D41D8CD98F00B204E9800998ECF8427E,config/settings.json
```

规则：

- 全量更新时，`path` 是文件相对于指定根目录的路径。
- 增量更新时，`path` 由调用方提供，并且必须使用与全量更新相同的相对路径表示。
- `path` 在所有平台上统一使用 `/` 作为目录分隔符，不得使用 `\`。访问文件系统时由实现转换为当前平台的本地路径。
- path 不得为空。
- 每行前 32 个字符是 MD5，第 33 个字符必须是逗号，剩余内容全部是 path。
- path 可以包含逗号，但不得包含回车或换行符。
- MD5 只计算文件内容，不包含 path 或其他数据。
- MD5 必须是 32 位大写十六进制字符串。
- 每个 path 只能出现一次。
- 记录按 path 原始字节升序排列，使文件内容保持稳定且便于查看差异。
- Manifest 本身不得作为待同步文件参与比较。
- 根目录下符合 `.file-md5-*.tmp` 的文件名保留给 Manifest 临时文件，不得作为待同步文件；嵌套目录中的同名文件不受此限制。

## 3. 文件 MD5

定义：

```text
fileMD5 = MD5Upper(file content bytes)
MD5Upper(data) = upper-case hexadecimal representation of MD5(data)
```

文件必须使用流式读取计算 MD5，不得为了计算 MD5 将整个文件加载到内存。

## 4. 对外函数

包只提供两个外部函数：

```go
func FullUpdate(rootDir string) error

func IncrementalUpdate(rootDir string, relativePath string) error
```

### `FullUpdate`

`rootDir` 是需要生成 Manifest 的根目录。函数递归遍历该目录下的普通文件，计算每个文件的内容 MD5，并使用相对于 `rootDir` 的路径生成完整 Manifest。

Manifest 固定为 `filepath.Join(rootDir, "file-md5.txt")`。

旧 Manifest 中存在、但在本次目录遍历中不存在的记录不会保留。

调用方必须在相关文件全部同步成功后调用该函数。

### `IncrementalUpdate`

`rootDir` 是文件所在的根目录，`relativePath` 是目标文件相对于 `rootDir` 的路径，并在所有平台上使用 `/` 作为目录分隔符。函数读取该文件并计算内容 MD5，然后新增或替换 Manifest 中对应 `relativePath` 的记录；其他旧记录保持不变。

单个文件更新示例：

```go
IncrementalUpdate(rootDir, "assets/app.tar.gz")
```

### 共同规则

- `FullUpdate` 遍历根目录中的全部文件；`IncrementalUpdate` 只读取 `relativePath` 指定的一个文件。
- 两个函数都在内部流式读取文件并计算大写 MD5，调用方不传入 MD5。
- 两个函数都从 `rootDir` 自动确定 Manifest 路径，调用方不传入 `manifestPath`。
- `relativePath` 为空或文件无法读取时返回错误。
- 任何目录遍历、文件读取、校验、读取旧 Manifest 或写入失败时返回错误，并保持原 Manifest 不变。
- 同一进程内，对同一个 Manifest 的调用会按 `rootDir` 自动串行执行；不同 Manifest 可以并发更新。跨进程调用仍需由调用方保证串行。
- 函数成功返回表示新的 Manifest 已经完成原子替换。

除这两个函数外，解析、排序和临时文件处理都应是包内实现。

对应 runner：

```text
full-update-file-manifest
payload: {"root_dir":"downloads"}

incremental-update-file-manifest
payload: {"root_dir":"downloads","relative_path":"example.txt"}
```

## 5. 比较流程

1. 读取 `file-md5.txt`，构建 `path -> lastSyncedMD5` 映射。
2. 遍历当前文件集合，计算每个文件的 `currentMD5`。
3. 按照以下规则分类：
   - Manifest 中不存在 path：新增文件。
   - Manifest 中存在 path，但 MD5 不同：内容发生变化。
   - Manifest 中存在 path，且 MD5 相同：没有变化，直接跳过。
4. Manifest 中存在、当前文件集合中不存在的 path，只会在下一次 `FullUpdate` 时从 Manifest 移除。

## 6. 同步和更新

新增或内容发生变化的文件需要重新同步。

处理规则：

1. 同步失败时保留 Manifest 中原有的 MD5。
2. 单个文件同步成功后，将其根目录和相对路径传给 `IncrementalUpdate`，由函数重新读取文件并更新 MD5。
3. 完整同步成功后，可以将根目录传给 `FullUpdate` 重新建立 Manifest。

## 7. 增量更新 Manifest

增量更新的输入是一个根目录和一个相对文件路径：

```text
filePath = filepath.Join(rootDir, relativePath)
```

函数先流式读取 `filePath` 并计算大写 MD5，再生成新的 `MD5,relativePath` 记录。随后使用以下方式更新 Manifest：

1. 逐行读取旧 Manifest，不需要把完整列表加载到内存。
2. path 小于 `relativePath` 的旧记录原样写入临时文件。
3. path 等于 `relativePath` 时，写入新的 `MD5,relativePath` 记录，替换旧记录。
4. path 大于 `relativePath` 且尚未写入新记录时，先写入新记录，再继续写入旧记录。
5. 遍历结束仍未写入新记录时，将其追加到末尾。
6. 完成后原子替换旧 Manifest。

增量更新必须完整读取一次旧 Manifest，时间复杂度为 `O(N)`，额外内存为 `O(1)`。

## 8. 全量更新 Manifest

全量更新忽略旧 Manifest 内容，完全根据指定根目录下的当前文件生成新的 Manifest：

1. 确认 `rootDir` 存在并且是目录。
2. 递归遍历 `rootDir` 下的普通文件，不跟随符号链接。
3. 对每个文件进行流式读取并计算大写 MD5。
4. 使用文件相对于 `rootDir` 的路径生成 `MD5,path` 记录，并将目录分隔符统一转换为 `/`。
5. 排除 Manifest 文件及根目录下符合 `.file-md5-*.tmp` 的保留临时文件，避免将其写入自身。
6. 按相对路径原始字节排序。
7. 将全部记录写入临时文件。
8. 原子替换旧 Manifest。

全量更新适用于首次生成 Manifest、明确要求重新建立基线，或者 Manifest 已无法使用的情况。

如果 Manifest 表示“最后一次成功同步”的状态，则全量更新前必须确保所有当前文件已经成功同步。任何文件同步失败时，都不得用当前文件集合直接覆盖旧 Manifest，否则会把未成功同步的文件错误地标记为已同步。

## 9. Manifest 持久化

更新 Manifest 时必须避免进程中断导致文件损坏：

1. 在 Manifest 所在目录创建临时文件。
2. 按 path 排序后写入全部记录。
3. 关闭临时文件并确认写入成功。
4. 使用原子 rename 替换原 Manifest。

增量更新时，如果 Manifest 不存在，按空记录处理；如果 Manifest 格式错误、MD5 非法或 path 重复，则停止处理并返回错误。全量更新不读取旧 Manifest，因此可以直接重建损坏的文件。

## 10. 验收场景

实现至少需要覆盖：

- Manifest 不存在时能够完成首次同步并创建文件。
- 文件内容未变化时不重复同步。
- 新增文件能够被发现并同步。
- 文件内容变化后能够重新同步。
- 同步失败时保留旧 MD5。
- `IncrementalUpdate` 能够根据根目录和相对路径计算 MD5，正确新增或替换一条记录，并保留其他记录。
- 增量更新只需流式读取旧 Manifest，不需要加载完整列表。
- `FullUpdate` 能够递归遍历指定目录，以相对路径生成完整 Manifest，并移除不存在文件的旧记录。
- 全量同步失败时不会错误地覆盖旧 Manifest。
- Manifest 更新过程中断时旧文件仍然可用。
- 输出记录始终按 path 排序。
- 所有 MD5 均为 32 位大写十六进制字符串。

MD5 仅用于非安全性的内容变化检测，不得用于对抗恶意碰撞的安全场景。
