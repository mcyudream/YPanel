# YPanel 生成：Go 多阶段构建（文件在 src 内，可自行修改；重建时选择保留则不覆盖）
FROM golang:{{.GO_VERSION}}-alpine AS build
WORKDIR /src
COPY . .
RUN go build -ldflags "-s -w" -o /out/app .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/app /app/app
ENV PORT={{.PORT}}
EXPOSE {{.PORT}}
CMD ["/app/app"]
