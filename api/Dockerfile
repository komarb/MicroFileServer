FROM golang:1.13

WORKDIR /app

COPY . .

RUN go build main.go
RUN chmod +x main

CMD [ "./main" ]
