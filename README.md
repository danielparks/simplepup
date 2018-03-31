# Simplepup — query PuppetDB using SSH for authentication

~~~
❯ ./simplepup 'facts[value] { name = "fqdn" limit 1 }'
[
  {
    "value": "vmware-statsfeeder5.ops.puppetlabs.net"
  }
]
~~~

Simplepup connects to the PuppetDB host over SSH, then queries the
unauthenticated HTTP endpoint.

Licensed under the [Simplified BSD License](LICENSE).
