FROM golang:1.8

COPY . .

RUN go-wrapper download

RUN go-wrapper install

EXPOSE 9140

CMD ["go-wrapper", "run"]

