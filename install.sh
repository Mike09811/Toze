#!/usr/bin/env bash
# Toze 快速安装：在全新 Ubuntu / Debian 服务器上一键部署控制面。
# 用法一（推荐）: curl -fsSL https://raw.githubusercontent.com/Mike09811/Toze/main/install.sh | bash
# 用法二:        git clone https://github.com/Mike09811/Toze.git && cd Toze && bash install.sh
set -euo pipefail

TOZE_DIR="${TOZE_DIR:-$(pwd)/Toze}"
SERVER_PORT="${SERVER_PORT:-8088}"
WEB_PORT="${WEB_PORT:-8080}"

log()  { printf '\033[36m[toze]\033[0m %s\n' "$*"; }
fail() { printf '\033[31m[toze]\033[0m %s\n' "$*" >&2; exit 1; }

rand_hex() { head -c "$1" /dev/urandom | od -An -tx1 | tr -d ' \n'; }

# 1) Docker
if ! command -v docker >/dev/null 2>&1; then
    log "安装 Docker ..."
    curl -fsSL https://get.docker.com | sh
fi
if ! docker compose version >/dev/null 2>&1; then
    fail "Docker Compose 不可用，请手动安装后重试"
fi
SUDO=""
[ "$(id -u)" = 0 ] || SUDO="sudo -n"
$SUDO docker version >/dev/null 2>&1 || fail "当前用户无法使用 docker（请把用户加入 docker 组后重新登录）"

# 2) 源码（curl 管道模式下克隆；已在仓库目录则跳过）
if [ ! -f docker-compose.yml ]; then
    command -v git >/dev/null 2>&1 || fail "缺少 git"
    log "克隆 Toze 仓库 ..."
    git clone --depth 1 https://github.com/Mike09811/Toze.git "$TOZE_DIR"
    cd "$TOZE_DIR"
fi
[ -f docker-compose.yml ] || fail "当前目录不是 Toze 仓库（缺少 docker-compose.yml）"

# 3) 生成 .env（已存在则保留，绝不覆盖密钥）
if [ ! -f .env ]; then
    log "生成 .env 密钥 ..."
    cat > .env <<EOF
DB_PASSWORD=$(rand_hex 12)
DB_NAME=toze
JWT_SECRET=$(rand_hex 32)
PAAS_AES_KEY=$(rand_hex 16)
EOF
    chmod 600 .env
fi

# 4) 构建并启动
log "构建并启动（首次需要几分钟拉取依赖与编译）..."
$SUDO docker compose up -d --build

# 5) 等待健康检查
log "等待服务就绪 ..."
for i in $(seq 1 60); do
    if curl -fsS "http://127.0.0.1:${SERVER_PORT}/healthz" >/dev/null 2>&1; then
        break
    fi
    sleep 3
    [ "$i" = 60 ] && fail "服务未在预期时间内就绪，请执行 docker compose logs server 排查"
done

IP="$(curl -fsS -4 --max-time 3 ifconfig.me 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}')"
printf '\n\033[32m✔ Toze 已启动\033[0m\n'
echo "  控制台:  http://${IP:-服务器IP}:${WEB_PORT}"
echo "  默认账号: admin / admin （请立即修改密码）"
echo "  数据目录: $(pwd) （.env 内含密钥，请妥善保管）"
