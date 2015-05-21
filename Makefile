all:
	goxc -bc='linux,darwin,windows,!386,!arm' -d bin xc

copy-dk5: all
	rm -fR /Users/austin_moran/devProjects/ssi/dk5-middle-tier/etc/oauth/*
	cp -fR bin/snapshot/* /Users/austin_moran/devProjects/ssi/dk5-middle-tier/etc/oauth/

clean:
	rm -fR bin/*

.PHONY:
	all clean
