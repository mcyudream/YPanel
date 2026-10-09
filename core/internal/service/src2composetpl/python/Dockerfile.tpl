# YPanel 生成：Python
FROM python:{{.PY_VERSION}}-alpine
WORKDIR /app
COPY . .
RUN if [ -f requirements.txt ]; then pip install --no-cache-dir -r requirements.txt; elif [ -f pyproject.toml ]; then pip install --no-cache-dir .; fi
ENV PORT={{.PORT}}
EXPOSE {{.PORT}}
CMD ["sh", "-c", "{{.START_CMD}}"]
