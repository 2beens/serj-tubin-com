FROM golang:latest

WORKDIR /go/src/app
COPY . .

ENV OPEN_WEATHER_API_KEY="dummy"
ENV SERJ_TUBIN_COM_ADMIN_USERNAME="dummy"
ENV SERJ_TUBIN_COM_ADMIN_PASSWORD_HASH="dummy"
ENV SERJ_BROWSER_REQ_SECRET="dummy"

RUN go build -o bin/service cmd/service/main.go

CMD ["./bin/service", "-env", "dockerdev"]
