ARG REPO
ARG TAG
FROM ${REPO}/k3os-kernel:${TAG} as kernel

ARG REPO
ARG TAG
FROM ${REPO}/k3os-iso:${TAG} as iso

ARG REPO
ARG TAG
FROM ${REPO}/k3os-gobuild:${TAG} as gobuild
ARG VERSION
ARG ARCH

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

RUN set -x \
 && apk add \
    libvirt \
    qemu-img
RUN [ "$ARCH" == "amd64" ] \
 && set -x \
 && apk add \
    grub-bios \
    ovmf \
    qemu-system-x86_64 \
 || true
RUN [ "$ARCH" == "arm64" ] \
 && set -x \
 && apk add \
    qemu-system-aarch64 \
 || true
RUN ln -s /usr/bin/qemu-system-* /usr/bin/qemu-system

COPY /pkg/ $GOPATH/src/github.com/rancher/k3os/pkg/
COPY /vendor/ $GOPATH/src/github.com/rancher/k3os/vendor/
COPY --from=kernel /output/ /output/
COPY --from=iso /output/ /output/
COPY images/07-qemu/*.sh /usr/local/bin/
WORKDIR /output
