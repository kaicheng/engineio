TESTS = test/server.js
REPORTER = dot

test:
	@go install $(RACE) github.com/kaicheng/engineio/test
	@./test/node_modules/.bin/mocha \
		--reporter $(REPORTER) \
		--timeout 10s \
		--bail \
		$(FILTER) $(TESTS)

.PHONY: test
