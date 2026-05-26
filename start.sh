#!/bin/bash

set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOG_DIR="/tmp/k8sgate"
mkdir -p "$LOG_DIR"

echo "======================================"
echo "    K8sGate 本地开发启动脚本"
echo "======================================"

# 1. 停止旧进程
echo ""
echo "[1/6] 清理旧进程..."
pkill -f "$PROJECT_DIR/backend/tmp/main" 2>/dev/null || true
pkill -f "node.*$PROJECT_DIR/frontend" 2>/dev/null || true
# 清理可能残留的 go run 进程
pgrep -f "go-build.*main" | xargs kill 2>/dev/null || true
sleep 1

# 2. 检查并启动 Docker
echo ""
echo "[2/6] 检查 Docker..."
if ! docker info &>/dev/null; then
    echo "  启动 Docker Desktop..."
    open -a Docker
    for i in $(seq 1 30); do
        if docker info &>/dev/null; then break; fi
        sleep 2
    done
    if ! docker info &>/dev/null; then
        echo "  [ERROR] Docker 启动超时"
        exit 1
    fi
fi
echo "  Docker 已就绪"

# 3. 检查并启动 MySQL
echo ""
echo "[3/6] 检查 MySQL..."
if ! docker ps --format '{{.Names}}' | grep -q k8sgate-mysql; then
    if docker ps -a --format '{{.Names}}' | grep -q k8sgate-mysql; then
        docker start k8sgate-mysql >/dev/null
    else
        echo "  创建 MySQL 容器..."
        docker run -d \
            --name k8sgate-mysql \
            -p 3306:3306 \
            -e MYSQL_ROOT_PASSWORD=root \
            -e MYSQL_DATABASE=k8sgate \
            mysql:5.7 \
            --character-set-server=utf8mb4 \
            --collation-server=utf8mb4_unicode_ci >/dev/null
    fi
    echo "  等待 MySQL 启动..."
    sleep 5
fi
echo "  MySQL 已就绪"

# 4. 设置环境变量（本地开发用）
echo ""
echo "[4/6] 设置环境变量..."
export DB_HOST=127.0.0.1
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=root
export DB_NAME=k8sgate
export JWT_SECRET=k8sgate-dev-secret
export SERVER_PORT=8080
export PASSWORD_MAX_AGE=90
export DASHBOARD_URL=http://localhost:8001/api/v1/namespaces/kubernetes-dashboard/services/https:kubernetes-dashboard:/proxy/
export GIN_MODE=debug

echo "  DB: ${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"

# 5. 检查 kubectl proxy
echo ""
echo "[5/6] 检查 kubectl proxy..."
if ! pgrep -f "kubectl proxy" &>/dev/null; then
    if command -v kubectl &>/dev/null; then
        kubectl proxy --port=8001 > "$LOG_DIR/kubectl-proxy.log" 2>&1 &
        sleep 2
        if curl -sf http://localhost:8001/healthz &>/dev/null; then
            echo "  kubectl proxy 已启动"
        else
            echo "  [WARN] kubectl proxy 启动失败，K8s 功能不可用"
        fi
    else
        echo "  [WARN] kubectl 未安装，跳过"
    fi
else
    echo "  kubectl proxy 已在运行"
fi

# 6. 启动后端
echo ""
echo "[6/6] 启动服务..."
cd "$PROJECT_DIR/backend"
go run main.go > "$LOG_DIR/backend.log" 2>&1 &
BACKEND_PID=$!
echo "$BACKEND_PID" > "$LOG_DIR/backend.pid"

for i in $(seq 1 20); do
    if curl -sf http://localhost:${SERVER_PORT}/healthz &>/dev/null; then
        echo "  后端启动成功 (PID: $BACKEND_PID)"
        break
    fi
    sleep 1
done
if ! curl -sf http://localhost:${SERVER_PORT}/healthz &>/dev/null; then
    echo "  [ERROR] 后端启动失败"
    tail -10 "$LOG_DIR/backend.log" 2>/dev/null
    exit 1
fi

# 启动前端
cd "$PROJECT_DIR/frontend"
npm run dev > "$LOG_DIR/frontend.log" 2>&1 &
FRONTEND_PID=$!
echo "$FRONTEND_PID" > "$LOG_DIR/frontend.pid"

for i in $(seq 1 15); do
    if curl -sf http://localhost:5173 &>/dev/null; then
        echo "  前端启动成功 (PID: $FRONTEND_PID)"
        break
    fi
    sleep 1
done
if ! curl -sf http://localhost:5173 &>/dev/null; then
    echo "  [ERROR] 前端启动失败"
    tail -5 "$LOG_DIR/frontend.log" 2>/dev/null
    exit 1
fi

echo ""
echo "======================================"
echo "  启动完成"
echo "======================================"
echo ""
echo "  前端: http://localhost:5173"
echo "  后端: http://localhost:${SERVER_PORT}"
echo ""
echo "  默认管理员: admin / Admin@123"
echo ""
echo "  日志: tail -f $LOG_DIR/backend.log"
echo "  停止: kill \$(cat $LOG_DIR/backend.pid) \$(cat $LOG_DIR/frontend.pid)"
