web/data.json: ./data/simonrim-apothecary.csv
	go run ./cmd/gen-data/ $< >$@~ 
	mv $@~ $@

.PHONY: serve
serve:
	go run ./cmd/serve ./web
