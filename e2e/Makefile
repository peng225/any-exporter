PROMBLOCK := ../promblock

.PHONY: test
test: $(PROMBLOCK)
	$(PROMBLOCK) &
	go test -v
	pkill $(notdir $(PROMBLOCK))

$(PROMBLOCK):
	make -C ../
