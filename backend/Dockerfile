# 使用 golang 1.21 作為基礎映像
FROM golang:1.21-alpine

# 設置工作目錄
WORKDIR /app/backend


# 安裝必要的系統套件
RUN apk add --no-cache git

# 複製 go.mod 和 go.sum
COPY go.mod ./
COPY go.sum ./

# 下載依賴
RUN go mod download

# 複製源代碼
COPY . .

# 編譯應用
RUN go build -o main cmd/main.go

# 暴露端口
EXPOSE 8080

# 運行應用
CMD ["./main"] 