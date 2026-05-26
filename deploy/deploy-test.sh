#!/bin/bash

# ================================================
# K8sGate 测试环境部署脚本（单节点，control 节点直接执行）
#
# 在 control 节点上直接执行，无需 SSH
#
# 架构：
#   后端：Go 二进制 + systemd
#   前端：npm run build -> podman nginx 容器
#   数据库：MySQL 本地
#   kubectl：直接可用（control 节点）
#
# 环境：Rocky Linux 9.7, K8s 已部署
# ================================================

set -e

# ==================== 配置 ====================
DEPLOY_DIR="/opt/k8sgate"

# 数据库
DB_HOST="127.0.0.1"
DB_PORT="3306"
DB_USER="root"
DB_PASSWORD="Admin@123"
DB_NAME="k8sgate_test"

# 端口
BACKEND_PORT="8080"
FRONTEND_PORT="80"

# nginx 镜像
NGINX_IMAGE="swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/nginx:latest"

# K8s Dashboard（将在部署时自动检测）
DASHBOARD_URL=""

JWT_SECRET="k8sgate-test-secret-2026"
# ==============================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# 验证包结构
if [ ! -f "$PROJECT_DIR/backend/k8sgate-backend" ]; then
    echo "[ERROR] 未找到 backend/k8sgate-backend"
    echo "  请确认包是通过 package.sh 构建的"
    exit 1
fi
if [ ! -d "$PROJECT_DIR/frontend/dist" ]; then
    echo "[ERROR] 未找到 frontend/dist/"
    echo "  请确认包是通过 package.sh 构建的"
    exit 1
fi

# 显示版本信息
if [ -f "$PROJECT_DIR/VERSION" ]; then
    echo "  版本信息:"
    cat "$PROJECT_DIR/VERSION" | while read -r line; do echo "    $line"; done
    echo ""
fi

echo "======================================"
echo "  K8sGate 测试环境部署（单节点）"
echo "======================================"
echo ""
echo "  部署目录: $DEPLOY_DIR"
echo "  前端:     nginx 容器 (port $FRONTEND_PORT)"
echo "  后端:     二进制 (port $BACKEND_PORT)"
echo ""

# ----------------------------------------
# 1. 环境检测
# ----------------------------------------
echo "[1/7] 环境检测..."

printf '  OS:      '; cat /etc/redhat-release 2>/dev/null || echo unknown
printf '  podman:  '; podman -v 2>/dev/null | awk '{print $3}' || echo missing
printf '  kubectl: '; kubectl version --client --short 2>/dev/null || kubectl version --client 2>/dev/null | head -1 || echo missing

# MySQL
echo "  -> MySQL ($DB_HOST)"
if mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p"${DB_PASSWORD}" -e "SELECT 1" &>/dev/null; then
    echo "     OK"
else
    echo "     [ERROR] MySQL 连接失败"
    exit 1
fi

# K8s 集群
echo "  -> K8s 集群"
kubectl get nodes --no-headers 2>/dev/null | while read -r line; do echo "     $line"; done

# Dashboard 自动检测
echo "  -> K8s Dashboard 检测"
DETECTED_DASHBOARD_URL=""
DASHBOARD_INFO=$(kubectl get svc -n kubernetes-dashboard --no-headers 2>/dev/null || true)

if echo "$DASHBOARD_INFO" | grep -q dashboard; then
    echo "     服务已安装"

    # 列出所有 dashboard 服务，方便排查
    echo "     所有服务:"
    kubectl get svc -n kubernetes-dashboard 2>/dev/null | while read -r line; do echo "       $line"; done

    # 检测服务名称（兼容 kubernetes-dashboard 和 kubernetes-dashboard-web 等）
    DASHBOARD_SVC_NAME=$(kubectl get svc -n kubernetes-dashboard --no-headers 2>/dev/null \
        | awk '{print $1}' | grep -E '^kubernetes-dashboard(-web)?$' | head -1)
    if [ -z "$DASHBOARD_SVC_NAME" ]; then
        DASHBOARD_SVC_NAME=$(kubectl get svc -n kubernetes-dashboard --no-headers 2>/dev/null \
            | awk '{print $1}' | head -1)
    fi
    echo "     选中服务: $DASHBOARD_SVC_NAME"

    # 获取服务详细信息
    SVC_TYPE=$(kubectl get svc "${DASHBOARD_SVC_NAME}" -n kubernetes-dashboard -o jsonpath='{.spec.type}' 2>/dev/null || true)
    SVC_PORT=$(kubectl get svc "${DASHBOARD_SVC_NAME}" -n kubernetes-dashboard -o jsonpath='{.spec.ports[0].port}' 2>/dev/null || true)
    SVC_PROTOCOL=$(kubectl get svc "${DASHBOARD_SVC_NAME}" -n kubernetes-dashboard -o jsonpath='{.spec.ports[0].name}' 2>/dev/null || true)
    SVC_TARGET_PORT=$(kubectl get svc "${DASHBOARD_SVC_NAME}" -n kubernetes-dashboard -o jsonpath='{.spec.ports[0].targetPort}' 2>/dev/null || true)
    echo "     类型: ${SVC_TYPE}, 端口: ${SVC_PORT}, 目标端口: ${SVC_TARGET_PORT}, 协议名: ${SVC_PROTOCOL}"

    # 判断协议
    SCHEME="https"
    if echo "$SVC_PROTOCOL" | grep -qi "^http$"; then
        SCHEME="http"
    fi
    # 如果端口是 80 或 8000 等常见 HTTP 端口，使用 http
    if [ "$SVC_PORT" = "80" ] || [ "$SVC_PORT" = "8000" ] || [ "$SVC_PORT" = "9090" ]; then
        SCHEME="http"
    fi

    if [ "$SVC_TYPE" = "NodePort" ]; then
        # NodePort 方式：通过本机端口访问
        NODE_PORT=$(kubectl get svc "${DASHBOARD_SVC_NAME}" -n kubernetes-dashboard -o jsonpath='{.spec.ports[0].nodePort}' 2>/dev/null || true)
        if [ -n "$NODE_PORT" ]; then
            DETECTED_DASHBOARD_URL="${SCHEME}://127.0.0.1:${NODE_PORT}"
            echo "     [NodePort] -> ${DETECTED_DASHBOARD_URL}"
        fi
    fi

    if [ -z "$DETECTED_DASHBOARD_URL" ]; then
        # ClusterIP 方式：control 节点可直接访问 ClusterIP
        CLUSTER_IP=$(kubectl get svc "${DASHBOARD_SVC_NAME}" -n kubernetes-dashboard -o jsonpath='{.spec.clusterIP}' 2>/dev/null || true)
        if [ -n "$CLUSTER_IP" ] && [ "$CLUSTER_IP" != "None" ]; then
            DETECTED_DASHBOARD_URL="${SCHEME}://${CLUSTER_IP}:${SVC_PORT}"
            echo "     [ClusterIP] -> ${DETECTED_DASHBOARD_URL}"
        fi
    fi

    # 验证连通性
    if [ -n "$DETECTED_DASHBOARD_URL" ]; then
        HTTP_CODE=$(curl -sk --connect-timeout 5 "${DETECTED_DASHBOARD_URL}" -o /dev/null -w '%{http_code}' 2>/dev/null || echo "000")
        echo "     连通性检测: HTTP ${HTTP_CODE}"
        if echo "$HTTP_CODE" | grep -qE '^[23]'; then
            echo "     连通性检测: OK"
        else
            echo "     [WARN] 连通性检测失败 (HTTP ${HTTP_CODE})，尝试其他方式..."

            # 尝试通过 K8s Service DNS（需要从 Pod 内部访问，control 节点可用 ClusterIP）
            # 再尝试所有 dashboard 服务
            for svc in $(kubectl get svc -n kubernetes-dashboard --no-headers 2>/dev/null | awk '{print $1}'); do
                svc_ip=$(kubectl get svc "$svc" -n kubernetes-dashboard -o jsonpath='{.spec.clusterIP}' 2>/dev/null || true)
                svc_port=$(kubectl get svc "$svc" -n kubernetes-dashboard -o jsonpath='{.spec.ports[0].port}' 2>/dev/null || true)
                if [ -n "$svc_ip" ] && [ "$svc_ip" != "None" ] && [ -n "$svc_port" ]; then
                    for proto in https http; do
                        test_code=$(curl -sk --connect-timeout 3 "${proto}://${svc_ip}:${svc_port}" -o /dev/null -w '%{http_code}' 2>/dev/null || echo "000")
                        echo "     尝试 ${svc} -> ${proto}://${svc_ip}:${svc_port} = HTTP ${test_code}"
                        if echo "$test_code" | grep -qE '^[23]'; then
                            DETECTED_DASHBOARD_URL="${proto}://${svc_ip}:${svc_port}"
                            echo "     [发现可用] -> ${DETECTED_DASHBOARD_URL}"
                            break 2
                        fi
                    done
                fi
            done
        fi
    fi
else
    echo "     [WARN] 未检测到 Dashboard 服务"
    echo "     检查 kubernetes-dashboard 命名空间是否存在:"
    kubectl get ns kubernetes-dashboard 2>/dev/null || echo "     命名空间不存在"
    echo "     所有命名空间:"
    kubectl get ns --no-headers 2>/dev/null | awk '{print "       " $1}'
fi

# 使用检测到的地址，或保留手动配置
if [ -n "$DETECTED_DASHBOARD_URL" ]; then
    DASHBOARD_URL="$DETECTED_DASHBOARD_URL"
    echo "     最终 DASHBOARD_URL=${DASHBOARD_URL}"
elif [ -z "$DASHBOARD_URL" ]; then
    DASHBOARD_URL="https://kubernetes-dashboard.kubernetes-dashboard.svc"
    echo "     [WARN] 未检测到可用地址，使用默认值: ${DASHBOARD_URL}"
fi

# 配置 Dashboard SA 权限（给 Dashboard 自身的 SA cluster-admin 权限）
# Dashboard v2 不使用 proxy 注入的 Bearer token，而是用自己的 SA 访问 API Server
# K8sGate 的访问控制在 proxy 层实现，所以这里放开 Dashboard SA 的权限是安全的
echo "  -> 配置 Dashboard ServiceAccount 权限..."
DASHBOARD_SA=$(kubectl get sa -n kubernetes-dashboard --no-headers 2>/dev/null | awk '{print $1}' | grep -E '^kubernetes-dashboard$' | head -1)
if [ -z "$DASHBOARD_SA" ]; then
    DASHBOARD_SA=$(kubectl get sa -n kubernetes-dashboard --no-headers 2>/dev/null | awk '{print $1}' | head -1)
fi

if [ -n "$DASHBOARD_SA" ]; then
    # 检查是否已有 ClusterRoleBinding
    EXISTING_CRB=$(kubectl get clusterrolebinding k8sgate-dashboard-admin --no-headers 2>/dev/null || true)
    if [ -z "$EXISTING_CRB" ]; then
        kubectl create clusterrolebinding k8sgate-dashboard-admin \
            --clusterrole=cluster-admin \
            --serviceaccount=kubernetes-dashboard:${DASHBOARD_SA} 2>/dev/null && \
            echo "     已创建 ClusterRoleBinding: k8sgate-dashboard-admin" || \
            echo "     [WARN] 创建 ClusterRoleBinding 失败"
    else
        echo "     ClusterRoleBinding 已存在"
    fi
    echo "     Dashboard SA: ${DASHBOARD_SA}"
else
    echo "     [WARN] 未找到 Dashboard ServiceAccount"
fi

# 配置 Dashboard 跳过登录（--enable-skip-login）
# Dashboard v2 有自己的 session 管理，不使用 proxy 注入的 Bearer token
# 需要跳过登录让 Dashboard 直接使用自己的 SA（已有 cluster-admin）访问 API Server
echo "  -> 配置 Dashboard 跳过登录..."
CURRENT_ARGS=$(kubectl get deploy kubernetes-dashboard -n kubernetes-dashboard \
    -o jsonpath='{.spec.template.spec.containers[0].args}' 2>/dev/null || true)

if echo "$CURRENT_ARGS" | grep -q "enable-skip-login"; then
    echo "     --enable-skip-login 已配置"
else
    echo "     添加 --enable-skip-login 和 --enable-insecure-login..."
    kubectl patch deploy kubernetes-dashboard -n kubernetes-dashboard --type='json' -p='[
        {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--enable-skip-login"},
        {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--enable-insecure-login"},
        {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--token-ttl=0"}
    ]' 2>/dev/null && echo "     已配置，Dashboard 将自动重启" || echo "     [WARN] patch 失败"
    # 等待 Dashboard 重启完成
    kubectl rollout status deploy kubernetes-dashboard -n kubernetes-dashboard --timeout=60s 2>/dev/null || true
fi

echo ""

# ----------------------------------------
# 2. 编译（如果有源码）
# ----------------------------------------
echo "[2/7] 验证构建产物..."

# 验证后端二进制是 linux/amd64
file "$PROJECT_DIR/backend/k8sgate-backend" | grep -q "ELF 64-bit" || {
    echo "  [ERROR] k8sgate-backend 不是 Linux ELF 二进制!"
    echo "  可能是 macOS 二进制，请用 package.sh 重新构建"
    exit 1
}
echo "  后端二进制: OK ($(du -sh "$PROJECT_DIR/backend/k8sgate-backend" | awk '{print $1}'))"
echo "  前端 dist:  OK ($(du -sh "$PROJECT_DIR/frontend/dist" | awk '{print $1}'))"

echo ""

# ----------------------------------------
# 3. 初始化数据库
# ----------------------------------------
echo "[3/7] 初始化数据库..."

mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p"${DB_PASSWORD}" \
    -e "CREATE DATABASE IF NOT EXISTS \`${DB_NAME}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

if [ -f "$SCRIPT_DIR/init.sql" ]; then
    sed "s/\`k8sgate\`/\`${DB_NAME}\`/g" "$SCRIPT_DIR/init.sql" | \
        mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p"${DB_PASSWORD}" ${DB_NAME} 2>/dev/null || true
    echo "  init.sql 已执行"
fi
echo ""

# ----------------------------------------
# 4. 停止旧服务 & 部署文件
# ----------------------------------------
echo "[4/7] 停止旧服务并部署文件..."

# 先停止旧服务，避免 "Text file busy"
systemctl stop k8sgate-frontend 2>/dev/null || true
systemctl stop k8sgate-backend 2>/dev/null || true
sleep 1
echo "  旧服务已停止"

mkdir -p ${DEPLOY_DIR}

# 始终拷贝构建产物到部署目录
cp -f "$PROJECT_DIR/backend/k8sgate-backend" "${DEPLOY_DIR}/"
rm -rf "${DEPLOY_DIR}/dist"
cp -r "$PROJECT_DIR/frontend/dist" "${DEPLOY_DIR}/dist"
cp -f "$SCRIPT_DIR/nginx.conf" "${DEPLOY_DIR}/nginx.conf" 2>/dev/null || true
# 拷贝版本信息
cp -f "$PROJECT_DIR/VERSION" "${DEPLOY_DIR}/VERSION" 2>/dev/null || true
echo "  构建产物已拷贝到 ${DEPLOY_DIR}/"

chmod +x "${DEPLOY_DIR}/k8sgate-backend"

# 生成环境变量文件
cat > "${DEPLOY_DIR}/.env" <<EOF
DB_HOST=${DB_HOST}
DB_PORT=${DB_PORT}
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}
JWT_SECRET=${JWT_SECRET}
DASHBOARD_URL=${DASHBOARD_URL}
SERVER_PORT=${BACKEND_PORT}
PASSWORD_MAX_AGE=90
GIN_MODE=release
EOF

echo "  .env 已生成, DASHBOARD_URL=${DASHBOARD_URL}"
echo ""

# ----------------------------------------
# 5. 生成 nginx 配置（确保 proxy_pass 指向 127.0.0.1）
# ----------------------------------------
echo "[5/7] 配置 nginx..."

cat > "${DEPLOY_DIR}/nginx.conf" <<'NGINX'
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    # index.html 禁止缓存，确保每次部署后用户拿到最新版本
    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        add_header Pragma "no-cache";
        add_header Expires "0";
    }

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_connect_timeout 10s;
        proxy_read_timeout 60s;
    }

    location /dashboard/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_connect_timeout 10s;
        proxy_read_timeout 300s;
    }
}
NGINX

echo "  nginx.conf 已生成 (proxy_pass -> 127.0.0.1:8080)"
echo ""

# ----------------------------------------
# 6. 拉取镜像 & 配置 systemd
# ----------------------------------------
echo "[6/7] 配置服务..."

# 拉取 nginx 镜像
if podman image exists "${NGINX_IMAGE}" 2>/dev/null; then
    echo "  nginx 镜像已存在"
else
    echo "  拉取 nginx 镜像..."
    podman pull "${NGINX_IMAGE}" 2>&1 | tail -1
fi

# 后端 systemd
cat > /etc/systemd/system/k8sgate-backend.service <<UNIT
[Unit]
Description=K8sGate Backend API
After=network.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${DEPLOY_DIR}
EnvironmentFile=${DEPLOY_DIR}/.env
ExecStart=${DEPLOY_DIR}/k8sgate-backend
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
UNIT

# 前端 nginx 容器 (systemd + podman, --network=host)
cat > /etc/systemd/system/k8sgate-frontend.service <<UNIT
[Unit]
Description=K8sGate Frontend (nginx)
After=network.target k8sgate-backend.service
Wants=k8sgate-backend.service

[Service]
Type=simple
ExecStartPre=-/usr/bin/podman rm -f k8sgate-frontend
ExecStart=/usr/bin/podman run --rm --name k8sgate-frontend \
    --network=host \
    -v ${DEPLOY_DIR}/dist:/usr/share/nginx/html:ro,Z \
    -v ${DEPLOY_DIR}/nginx.conf:/etc/nginx/conf.d/default.conf:ro,Z \
    ${NGINX_IMAGE}
ExecStop=/usr/bin/podman stop k8sgate-frontend
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable k8sgate-backend k8sgate-frontend 2>/dev/null
echo "  systemd 服务已配置"
echo ""

# ----------------------------------------
# 7. 启动服务并验证
# ----------------------------------------
echo "[7/7] 启动服务..."

# 启后端
echo "  -> 启动后端..."
systemctl start k8sgate-backend
sleep 3
if curl -sf http://localhost:${BACKEND_PORT}/healthz &>/dev/null; then
    echo "     Backend:  OK (port ${BACKEND_PORT})"
else
    echo "     Backend:  FAILED"
    journalctl -u k8sgate-backend --no-pager -n 10
fi

# 启前端
echo "  -> 启动前端..."
systemctl start k8sgate-frontend
sleep 3
if curl -sf http://localhost:${FRONTEND_PORT}/ &>/dev/null; then
    echo "     Frontend: OK (port ${FRONTEND_PORT})"
else
    echo "     Frontend: FAILED"
    journalctl -u k8sgate-frontend --no-pager -n 10
fi

# 验证 Dashboard 代理
echo "  -> 验证 Dashboard 代理..."
DASH_CODE=$(curl -sk --connect-timeout 5 -o /dev/null -w '%{http_code}' "http://localhost:${BACKEND_PORT}/healthz" 2>/dev/null || echo "000")
echo "     后端健康检查: HTTP ${DASH_CODE}"

# 测试 Dashboard 连通性（从后端视角）
echo "  -> 后端到 Dashboard 连通性:"
echo "     DASHBOARD_URL=${DASHBOARD_URL}"
DASH_DIRECT=$(curl -sk --connect-timeout 5 -o /dev/null -w '%{http_code}' "${DASHBOARD_URL}" 2>/dev/null || echo "000")
echo "     直接访问 Dashboard: HTTP ${DASH_DIRECT}"

# 防火墙
if systemctl is-active firewalld &>/dev/null; then
    firewall-cmd --permanent --add-port=${BACKEND_PORT}/tcp 2>/dev/null || true
    firewall-cmd --permanent --add-port=${FRONTEND_PORT}/tcp 2>/dev/null || true
    firewall-cmd --reload 2>/dev/null || true
    echo "  防火墙规则已添加"
fi

# 获取本机 IP
LOCAL_IP=$(hostname -I 2>/dev/null | awk '{print $1}')

echo ""
echo "======================================"
echo "  部署完成"
echo "======================================"
echo ""
echo "  访问地址:"
echo "    http://${LOCAL_IP:-localhost}"
echo ""
echo "  默认管理员: admin / Admin@123"
echo ""
echo "  Dashboard URL: ${DASHBOARD_URL}"
echo ""
echo "  服务管理:"
echo "    systemctl {status|restart|stop} k8sgate-backend"
echo "    systemctl {status|restart|stop} k8sgate-frontend"
echo "    journalctl -u k8sgate-backend -f"
echo "    journalctl -u k8sgate-frontend -f"
echo ""
echo "  排查 Dashboard:"
echo "    cat ${DEPLOY_DIR}/.env | grep DASHBOARD"
echo "    curl -sk ${DASHBOARD_URL}"
echo "    kubectl get svc -n kubernetes-dashboard"
echo "    podman logs k8sgate-frontend"
echo ""
