### BASE ###
ARG REPO
ARG TAG
FROM ${REPO}/k3os-progs:${TAG} as progs

FROM scratch
COPY --from=progs /output/k3os /sbin/
ENTRYPOINT ["k3os"]
CMD ["--help"]
