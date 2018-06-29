FROM busybox

WORKDIR /root/transform
COPY ./app/app ./app 
ENTRYPOINT [ "./app/app" ]