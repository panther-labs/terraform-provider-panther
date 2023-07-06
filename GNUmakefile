default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
ifndef PANTHER_API_URL
$(error PANTHER_API_URL is undefined)
endif
ifndef PANTHER_API_TOKEN
$(error PANTHER_API_TOKEN is undefined)
endif
	PANTHER_API_URL=${PANTHER_API_URL} PANTHER_API_TOKEN=${PANTHER_API_TOKEN} TF_ACC=1 go test ./internal/... -v -timeout 120m
