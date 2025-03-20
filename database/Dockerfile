# 使用 MySQL 8.0 作為基礎映像
FROM mysql:8.0

# 設置環境變數
ENV MYSQL_ROOT_PASSWORD=root
ENV MYSQL_DATABASE=vocabulary
ENV MYSQL_USER=vocab_user
ENV MYSQL_PASSWORD=vocab_pass

# 複製初始化腳本
COPY init.sql /docker-entrypoint-initdb.d/

# 設置字符集和排序規則
RUN echo "character-set-server=utf8mb4" >> /etc/mysql/conf.d/docker.cnf
RUN echo "collation-server=utf8mb4_unicode_ci" >> /etc/mysql/conf.d/docker.cnf

# 暴露端口
EXPOSE 3306 