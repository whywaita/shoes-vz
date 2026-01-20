#!/bin/bash
# VM 内で実行するセットアップスクリプト
# shoes-vz テスト用最小イメージ作成
set -euo pipefail

echo "=== Enabling SSH ==="
sudo systemsetup -setremotelogin on

echo "=== Creating runner user ==="
# runnerユーザーが存在するかチェック
if id runner &>/dev/null; then
    echo "Runner user already exists, ensuring home directory is set up correctly..."
else
    # sysadminctl を使ってユーザー作成（macOS 10.10以降）
    # -addUser: ユーザー名
    # -fullName: フルネーム
    # -password: パスワード（平文）
    # -home: ホームディレクトリ（デフォルトは /Users/username）
    # -admin: 管理者権限を付与
    echo "Creating runner user with sysadminctl..."
    sudo sysadminctl -addUser runner \
      -fullName "GitHub Actions Runner" \
      -password "runner" \
      -admin \
      -home /Users/runner
fi

# ホームディレクトリが存在しない場合は作成
if [ ! -d /Users/runner ]; then
    echo "Creating home directory for runner user..."
    sudo mkdir -p /Users/runner
fi

# NFSHomeDirectoryが正しく設定されているか確認
CURRENT_HOME=$(sudo dscl . -read /Users/runner NFSHomeDirectory 2>/dev/null | awk '{print $2}')
if [ "$CURRENT_HOME" != "/Users/runner" ]; then
    echo "Setting NFSHomeDirectory to /Users/runner..."
    sudo dscl . -create /Users/runner NFSHomeDirectory /Users/runner
fi

# パーミッション設定
echo "Setting permissions for home directory..."
sudo chown -R runner:staff /Users/runner
sudo chmod 755 /Users/runner

echo "Runner user setup completed successfully"

# sudoers 設定（パスワードなしで sudo 可能にする）
echo "=== Configuring sudoers for runner user ==="
echo "runner ALL=(ALL) NOPASSWD:ALL" | sudo tee /etc/sudoers.d/runner
sudo chmod 440 /etc/sudoers.d/runner

echo "=== Setting up SSH for runner user ==="
sudo -u runner mkdir -p /Users/runner/.ssh
sudo -u runner chmod 700 /Users/runner/.ssh

# SSH キーがある場合は authorized_keys に追加
if [ -f /tmp/ssh_public_key ]; then
    sudo -u runner cp /tmp/ssh_public_key /Users/runner/.ssh/authorized_keys
    sudo -u runner chmod 600 /Users/runner/.ssh/authorized_keys
    echo "SSH public key installed for runner user"
fi

echo "=== Installing Homebrew and basic tools as runner user ==="
# Homebrew のインストールと基本ツールのインストール（runnerユーザで実行）
# -H オプションで HOME 環境変数を runner のホームディレクトリに設定
sudo -H -u runner bash << 'HOMEBREW_SCRIPT'
# Homebrew のインストール（非対話的）
NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Homebrew の PATH 設定
echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> /Users/runner/.zprofile
eval "$(/opt/homebrew/bin/brew shellenv)"

echo "=== Installing basic tools ==="
brew install git curl wget jq yq
HOMEBREW_SCRIPT

echo "=== Installing shoes-vz-runner-agent ==="
if [ -f /tmp/shoes-vz-runner-agent ]; then
    sudo mkdir -p /usr/local/bin
    sudo mv /tmp/shoes-vz-runner-agent /usr/local/bin/
    sudo chmod +x /usr/local/bin/shoes-vz-runner-agent
    echo "shoes-vz-runner-agent installed"
else
    echo "Warning: shoes-vz-runner-agent not found at /tmp/shoes-vz-runner-agent"
fi

echo "=== Setting up runner-monitor LaunchDaemon ==="
# LaunchDaemonとして配置（ログイン不要でシステム起動時に実行）

# plistファイルを作成
# shoes-vz-runner-agentは起動時に.runnerファイルを監視し、
# runner IDが見つかったら自動的にIP通知を行う
sudo tee /Library/LaunchDaemons/com.github.whywaita.shoes-vz-runner-agent.plist > /dev/null << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.github.whywaita.shoes-vz-runner-agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/shoes-vz-runner-agent</string>
        <string>-listen</string>
        <string>:8080</string>
        <string>-runner-path</string>
        <string>/tmp/runner</string>
        <string>-host-ip</string>
        <string>192.168.64.1</string>
        <string>-agent-port</string>
        <string>8081</string>
    </array>
    <key>UserName</key>
    <string>runner</string>
    <key>GroupName</key>
    <string>staff</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/runner-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/runner-agent.error.log</string>
</dict>
</plist>
EOF

sudo chmod 644 /Library/LaunchDaemons/com.github.whywaita.shoes-vz-runner-agent.plist
echo "LaunchDaemon plist created at /Library/LaunchDaemons/"
echo "shoes-vz-runner-agent will start at boot time (no login required)"
echo "It will run as 'runner' user and automatically watch for .runner file and notify IP when runner is registered"

echo "=== Cleanup ==="
# キャッシュ削除（エラーは無視）
sudo rm -rf /Library/Caches/* 2>/dev/null || true
sudo rm -rf /Users/runner/Library/Caches/* 2>/dev/null || true

# ログ削除（エラーは無視）
sudo rm -rf /var/log/* 2>/dev/null || true
sudo rm -rf /Users/runner/Library/Logs/* 2>/dev/null || true

# 履歴削除
sudo rm -f /Users/runner/.bash_history /Users/runner/.zsh_history

# Spotlight 無効化（起動高速化）
sudo mdutil -a -i off

echo "=== Setup complete ==="
echo "You can now shutdown the VM with: sudo shutdown -h now"
