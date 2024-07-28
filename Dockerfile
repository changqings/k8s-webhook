FROM ubuntu:24.04

COPY k8s-webhook /usr/local/bin/

RUN chmod +x /usr/local/bin/k8s-webhook

CMD ["k8s-webhook"]
