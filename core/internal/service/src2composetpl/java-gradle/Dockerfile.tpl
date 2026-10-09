# YPanel 生成：Java Gradle 构建（多模块时 JAR_DIR 指向模块构建产物）
FROM gradle:8-jdk{{.JAVA_VERSION}} AS build
WORKDIR /src
COPY . .
RUN gradle -q bootJar --no-daemon || gradle -q build --no-daemon

FROM eclipse-temurin:{{.JAVA_VERSION}}-jre-alpine
WORKDIR /app
COPY --from=build /src/{{.JAR_DIR}}/build/libs/ /app/libs/
ENV PORT={{.PORT}} JAVA_OPTS=""
EXPOSE {{.PORT}}
CMD ["sh", "-c", "java $JAVA_OPTS -jar $(ls /app/libs/*.jar | grep -v plain | head -1)"]
