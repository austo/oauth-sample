DK5_PATH=/Users/austin_moran/devProjects/ssi/dk5-middle-tier/etc/oauth

all:
	goxc -bc='linux,darwin,windows,!386,!arm' -d bin xc

copy-dk5: all
	rm -fR $(DK5_PATH)/*
	cp -fR bin/snapshot/* $(DK5_PATH)/

clean:
	rm -fR bin/*

run:
	bin/snapshot/darwin_amd64/oauth-sample

.PHONY:
	all clean
