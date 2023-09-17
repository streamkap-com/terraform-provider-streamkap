default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 STREAMKAP_HOST=https://api.streamkap.com STREAMKAP_CLIENT_ID=client_id STREAMKAP_SECRET=secret go test ./... -v $(TESTARGS) -timeout 120m
