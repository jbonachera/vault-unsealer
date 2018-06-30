build::
	docker build -t jbonachera/vault-unsealer .
	docker create --name artifacts jbonachera/vault-unsealer
	docker cp artifacts:/bin/vault-unsealer vault-unsealer
	docker rm artifacts

