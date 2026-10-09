# YPanel 生成：Java Maven 构建（多模块时 -pl 定向构建目标模块及依赖；aliyun mirror 加速）
FROM maven:3.9-eclipse-temurin-{{.JAVA_VERSION}} AS build
WORKDIR /src
RUN mkdir -p /root/.m2 && echo '<settings><mirrors><mirror><id>aliyun</id><mirrorOf>central</mirrorOf><url>https://maven.aliyun.com/repository/public</url></mirror></mirrors></settings>' > /root/.m2/settings.xml
COPY . .
RUN mvn -q -DskipTests package{{.MAVEN_MODULE_ARGS}}

FROM eclipse-temurin:{{.JAVA_VERSION}}-jre-alpine
WORKDIR /app
COPY --from=build /src/{{.JAR_DIR}}/target/*.jar /app/app.jar
ENV PORT={{.PORT}} JAVA_OPTS=""
EXPOSE {{.PORT}}
CMD ["sh", "-c", "java $JAVA_OPTS -jar /app/app.jar"]
