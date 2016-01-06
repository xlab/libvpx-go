all:
	cgogen vpx.yml

clean:
	rm -f vpx/cgo_helpers.go vpx/cgo_helpers.h vpx/const.go vpx/doc.go vpx/types.go
	rm -f vpx/vpx.go

test:
	cd vpx && go build
	