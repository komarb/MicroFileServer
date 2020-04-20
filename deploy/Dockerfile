FROM alpine

RUN apk update \
        && apk upgrade \
        && apk add --no-cache \
        ca-certificates \
        && update-ca-certificates 2>/dev/null || true

WORKDIR /app

COPY MicroFileServer/microfileserver-api-build/main ./main

RUN chmod +x main

CMD [ "./main" ]