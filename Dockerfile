FROM golang

COPY ./cli/cli  /bin

EXPOSE 9119
CMD ["cli"]
