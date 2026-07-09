# Hướng dẫn triển khai Docker (Legacy / chưa xác minh)

Triển khai Docker hiện là đường dẫn legacy kế thừa từ dự án upstream và chưa được xác minh như một phương thức triển khai được hỗ trợ chính thức cho AgenticDub. Các đường dẫn được hỗ trợ chính hiện là Go server cục bộ, CLI, desktop app và MCP server.

Các ví dụ bên dưới chỉ được giữ lại để tham khảo. Không xem tên image đã xuất bản hoặc các đoạn compose là hướng dẫn phát hành hiện tại của AgenticDub cho đến khi Docker support được làm lại và xác minh.

## Bắt đầu nhanh
Trước tiên, chuẩn bị tệp cấu hình, thiết lập cổng lắng nghe của máy chủ là `8888`, địa chỉ lắng nghe của máy chủ là `0.0.0.0`.

### Khởi động bằng docker run
```bash
docker run -d \
  -p 8888:8888 \
  -v /path/to/config.toml:/app/config/config.toml \
  -v /path/to/tasks:/app/tasks \
  asteria798/krillinai
```

### Khởi động bằng docker-compose
```yaml
version: '3'
services:
  krillin:
    image: asteria798/krillinai
    ports:
      - "8888:8888"
    volumes:
      - /path/to/config.toml:/app/config/config.toml # Tệp cấu hình
      - /path/to/tasks:/app/tasks # Thư mục đầu ra
```

## Lưu trữ mô hình
Nếu sử dụng mô hình fasterwhisper, AgenticDub sẽ tự động tải xuống các tệp cần thiết cho mô hình vào thư mục `/app/models` và thư mục `/app/bin`. Sau khi xóa container, các tệp này sẽ bị mất. Nếu cần lưu trữ mô hình, bạn có thể ánh xạ hai thư mục này đến thư mục của máy chủ.

### Khởi động bằng docker run
```bash
docker run -d \
  -p 8888:8888 \
  -v /path/to/config.toml:/app/config/config.toml \
  -v /path/to/tasks:/app/tasks \
  -v /path/to/models:/app/models \
  -v /path/to/bin:/app/bin \
  asteria798/krillinai
```

### Khởi động bằng docker-compose
```yaml
version: '3'
services:
  krillin:
    image: asteria798/krillinai
    ports:
      - "8888:8888"
    volumes:
      - /path/to/config.toml:/app/config/config.toml      
      - /path/to/tasks:/app/tasks
      - /path/to/models:/app/models
      - /path/to/bin:/app/bin
```

## Lưu ý
1. Nếu chế độ mạng của container docker không phải là host, nên thiết lập địa chỉ lắng nghe của máy chủ trong tệp cấu hình là `0.0.0.0`, nếu không có thể không truy cập được dịch vụ.
2. Nếu trong container cần truy cập proxy mạng của máy chủ, hãy thiết lập mục cấu hình proxy `127.0.0.1` thành `host.docker.internal`, ví dụ `http://host.docker.internal:7890`.
