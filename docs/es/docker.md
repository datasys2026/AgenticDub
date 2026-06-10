# Guía de implementación de Docker (Legacy / no verificada)

La implementación con Docker es actualmente una ruta legacy heredada del proyecto upstream y todavía no se ha verificado como un método oficialmente soportado para AgenticDub. Las rutas principales soportadas son el Go server local, la CLI, la desktop app y el MCP server.

Los ejemplos siguientes se conservan solo como referencia. No trates el nombre de imagen publicado ni los fragmentos compose como instrucciones actuales de release de AgenticDub hasta que el Docker support se reconstruya y se verifique.

## Comenzar rápidamente
Primero, prepara el archivo de configuración, configurando el puerto de escucha del servidor en `8888` y la dirección de escucha del servidor en `0.0.0.0`.

### Inicio con docker run
```bash
docker run -d \
  -p 8888:8888 \
  -v /path/to/config.toml:/app/config/config.toml \
  -v /path/to/tasks:/app/tasks \
  asteria798/krillinai
```

### Inicio con docker-compose
```yaml
version: '3'
services:
  krillin:
    image: asteria798/krillinai
    ports:
      - "8888:8888"
    volumes:
      - /path/to/config.toml:/app/config/config.toml # Archivo de configuración
      - /path/to/tasks:/app/tasks # Directorio de salida
```

## Persistencia del modelo
Si utilizas el modelo fasterwhisper, AgenticDub descargará automáticamente los archivos necesarios para el modelo en el directorio `/app/models` y el directorio `/app/bin`. Estos archivos se perderán al eliminar el contenedor. Si necesitas persistir el modelo, puedes mapear estos dos directorios a un directorio en el host.

### Inicio con docker run
```bash
docker run -d \
  -p 8888:8888 \
  -v /path/to/config.toml:/app/config/config.toml \
  -v /path/to/tasks:/app/tasks \
  -v /path/to/models:/app/models \
  -v /path/to/bin:/app/bin \
  asteria798/krillinai
```

### Inicio con docker-compose
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

## Consideraciones
1. Si el modo de red del contenedor de Docker no es host, se recomienda configurar la dirección de escucha del servidor en el archivo de configuración como `0.0.0.0`, de lo contrario, es posible que no se pueda acceder al servicio.
2. Si el contenedor necesita acceder al proxy de red del host, configura la opción de dirección del proxy `proxy` de `127.0.0.1` a `host.docker.internal`, por ejemplo, `http://host.docker.internal:7890`.
