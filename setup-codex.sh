#!/bin/bash

# Setup codex in Cloud Shell

# 检查参数
if [ $# -lt 1 ]; then
    echo "用法: $0 <token>"
    exit 1
fi

rm -rf ~/.codex

npm install -g @openai/codex

# 定义目标文件路径
TARGET_DIR="$HOME/.codex"
AUTH_FILE="$TARGET_DIR/auth.json"

# 需要写入的 JSON 内容（示例）
JSON_CONTENT='{
  "OPENAI_API_KEY": "'$1'"
}'

# 或者如果你想写入纯文本（非JSON格式），可以这样定义：
# TEXT_CONTENT="这里是普通的文本内容"

# 创建目录（如果不存在），-p 参数可以递归创建且不报错
mkdir -p "$TARGET_DIR"

# 检查目录是否创建成功
if [ ! -d "$TARGET_DIR" ]; then
    echo "错误：无法创建目录 $TARGET_DIR"
    exit 1
fi

echo "$JSON_CONTENT" > "$AUTH_FILE"

# 检查文件是否写入成功
if [ $? -eq 0 ]; then
    echo "成功写入文件：$AUTH_FILE"
    # 可选：显示文件内容
    cat "$AUTH_FILE"
else
    echo "错误：写入文件失败"
    exit 1
fi

CONFIG_TOML_FILE="$TARGET_DIR/config.toml"

TOML_CONTENT='model_provider = "aicodemirror"
model = "gpt-5.4"
model_reasoning_effort = "xhigh"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.aicodemirror]
name = "aicodemirror"
base_url = "https://api.aicodemirror.com/api/codex/backend-api/codex"
wire_api = "responses"
'

echo "$TOML_CONTENT" > "$CONFIG_TOML_FILE"

# 检查文件是否写入成功
if [ $? -eq 0 ]; then
    echo "成功写入文件：$CONFIG_TOML_FILE"
    # 可选：显示文件内容
    cat "$CONFIG_TOML_FILE"
else
    echo "错误：写入文件失败"
    exit 1
fi