.PHONY: build-backend build-frontend build-all docker-backend docker-frontend docker-all deploy

# 镜像仓库地址，按实际修改
REGISTRY ?= registry.wanfangdata.com.cn/k8sgate
VERSION ?= latest

# ========== 本地编译 ==========

# 编译后端（交叉编译为 Linux amd64，适配 CentOS 7 部署）
build-backend:
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o k8sgate-backend ./main.go

# 编译前端
build-frontend:
	cd frontend && npm run build

build-all: build-backend build-frontend

# ========== Docker 镜像（先本地编译，再打包）==========

docker-backend: build-backend
	cd backend && docker build --platform linux/amd64 -t $(REGISTRY)/backend:$(VERSION) .

docker-frontend: build-frontend
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

# ========== 清理 ==========

clean:
	rm -f backend/k8sgate-backend
	rm -rf frontend/dist

# ========== 开发模式 ==========

dev-backend:
	cd backend && go run main.go

dev-frontend:
	cd frontend && npm run dev
