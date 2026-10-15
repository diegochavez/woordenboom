FROM golang:1.23-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
COPY cmd ./cmd

RUN go build -o /out/api ./cmd/api
RUN go build -o /out/import-opentaal ./cmd/import-opentaal
RUN go build -o /out/import-wordfreq ./cmd/import-wordfreq
RUN go build -o /out/classify-vocabulary ./cmd/classify-vocabulary
RUN go build -o /out/classify-typesafe ./cmd/classify-typesafe
RUN go build -o /out/report-candidates ./cmd/report-candidates
RUN go build -o /out/generate-word-batch ./cmd/generate-word-batch
RUN go build -o /out/export-word-batch ./cmd/export-word-batch
RUN go build -o /out/import-word-content ./cmd/import-word-content
RUN go build -o /out/prepare-content-workqueue ./cmd/prepare-content-workqueue

FROM alpine:3.20

RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/import-opentaal /app/import-opentaal
COPY --from=build /out/import-wordfreq /app/import-wordfreq
COPY --from=build /out/classify-vocabulary /app/classify-vocabulary
COPY --from=build /out/classify-typesafe /app/classify-typesafe
COPY --from=build /out/report-candidates /app/report-candidates
COPY --from=build /out/generate-word-batch /app/generate-word-batch
COPY --from=build /out/export-word-batch /app/export-word-batch
COPY --from=build /out/import-word-content /app/import-word-content
COPY --from=build /out/prepare-content-workqueue /app/prepare-content-workqueue

USER app

EXPOSE 8080

CMD ["/app/api"]
