.PHONY: build-backend build-frontend build-all docker-backend docker-frontend docker-all deploy

# 镜像仓库地址，按实际修改
REGISTRY ?= registry.wanfangdata.com.cn/k8sgate
VERSION ?= latest

# ========== 本地编译 ==========

# 编译后端（交叉编译为 Linux amd64，适配 CentOS 7 部署）
build-backend:
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../bin/k8sgate-backend ./main.go

# 编译前端
build-frontend:
	cd frontend && npm run build

build-all: build-backend build-frontend

# ========== Docker 镜像 ==========

# 构建后端镜像（多架构，适配 Linux amd64）
docker-backend:
	cd backend && docker build --platform linux/amd64 -t $(REGISTRY)/backend:$(VERSION) .

# 构建前端镜像
docker-frontend:
	cd frontend && docker build --platform linux/amd64 -t $(REGISTRY)/frontend:$(VERSION) .

docker-all: docker-backend docker-frontend

# ========== 推送镜像 ==========

push-backend:
	docker push $(REGISTRY)/backend:$(VERSION)

push-frontend:
	docker push $(REGISTRY)/frontend:$(VERSION)

push-all: push-backend push-frontend

# ========== 部署到 K8s ==========

deploy:
	kubectl apply -f deploy/namespace.yaml
	kubectl apply -f deploy/rbac.yaml
	kubectl apply -f deploy/secret.yaml
	kubectl apply -f deploy/backend.yaml
	kubectl apply -f deploy/frontend.yaml
	kubectl apply -f deploy/virtualservice.yaml

# ========== 开发模式 ==========

# 本地运行后端
dev-backend:
	cd backend && go run main.go

# 本地运行前端
dev-frontend:
	cd frontend && npm run dev
