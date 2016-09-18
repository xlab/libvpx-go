all:
	cgogen vpx.yml

clean:
	rm -f vpx/cgo_helpers.go vpx/cgo_helpers.h cgo_helpers.c
	rm -f vpx/const.go vpx/doc.go vpx/types.go
	rm -f vpx/vpx.go

test:
	cd vpx && go build
	