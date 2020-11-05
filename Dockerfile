FROM gcr.io/distroless/base-debian10

COPY ./bin/linux /aerial

ENTRYPOINT ["/aerial"]