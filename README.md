# shroud - Public Hidden Services

Shroud provides a mechanism to run a website that is accessible to any client connected to the internet but whose IP address (and thus network provider and location) is anonymized completely and can not be discovered by any intermediary. Henceforth in this document, I will refer to a service hidden by shroud as a **shrouded service**.

#### So it's like Tor hidden services?

Yes. Shrouded services are just like a Tor hidden service except that:

1. Unlike a Tor hidden service, clients do not need to run Tor to access a shrouded service.
1. Shrouded services do not have onion addresses, but just regular DNS-based hostnames. To a client, they look like any normal web service.
1. Shrouded services typically have lower latency than Tor hidden services.

Even though shroud does not use the Tor hidden service protocol, it *does* rely on Tor for providing anonymity to shrouded services.

## Download

Pre-compiled binaries are available for all three major operatings systems. 

- [Linux](http://dl.shroud.io/linux_386/dev/shroud.zip)
- [OS X](http://dl.shroud.io/darwin_amd64/dev/shroud.zip)
- [Windows](http://dl.shroud.io/windows_386/dev/shroud.zip) (requires separate tor installation)

#### DISCLAIMER
*THESE DOWNLOADS ARE FOR EXPERIMENTAL PURPOSES ONLY*.

- All of these binaries are delivered insecurely over HTTP.
- They are not signed.
- Shroud is an alpha-stage project and may contain serious flaws.
- Shroud has not been audited or peer-reviewed.

You should compile a client from source yourself!

## How do I run a shrouded service?

Running a shrouded service is pretty easy! To run a shrouded service on port 5555 for example.com, here are the steps:

1. With your DNS provider, create a CNAME record from example.com -> example.com.proxy.v1.shroud.io
1. Download the shroud client and run:

    ./shroud example.com 5555

This will begin forwarding TLS connections made to example.com to a service on port 5555 of your local machine. 

#### Generate a TLS key and certificate for example.com

You'll probably want to be able to decrypt those requests to do something useful with them though! To do that, you'll need a TLS key and certificate. You generate the key yourself, and then you can either self-sign your certificate or you can submit a signing request to a CA to get a certificate that browsers won't pop up annoying 'not trusted' dialogues for.

The easiest way to do this is with openssl. First create a key:

    openssl genrsa -out example.com.key 2048

Then if you want to self-sign your certificate, you can do that with:

    openssl -x509 -new -nodes -key example.com.key -days 3650 -out example.com.crt
    
If you want to submit it to a CA to have it signed, you'll do this instead to create the CSR (certificate signing request) that a CA will ask you for:

    openssl req -new -key example.com.key -out example.com.csr

#### Okay, I have a key and certificate, now what?

Now, instead of running shroud like this:

    ./shroud example.com 5555

You'll pass it your key and certificate file so that it can terminate your TLS traffic right before it hands it off to your service:

    ./shroud -tlsKey=/path/to/example.com.key -tlsCrt=/path/to/example.com.crt example.com 5555

If the service you're running on port 5555 is capable of terminating TLS traffic itself (like Apache or nginx), it's recommended that you just use that functionality to terminate your connections.

## What do shrouded services *NOT* provide?

1. Shrouded services provide no anonymity for any clients accessing the service.
1. Shrouded services rely on DNS for discoverability. This means a shrouded service can become unavailable if the DNS provider or registrar revokes your domain.
1. Shrouded services rely on public proxies to bridge between Tor and the public internet. These proxies could be taken down by a service provider or DDOS attacks.
1. The location of a shrouded service may be discoverable to a global adversary.
1. Shrouded services are *not* accessible by browsers which do not support the TLS SNI extension, namely all versions of Internet Explorer on Windows XP and the default browser on Android 2.\*.

## How does it work?

shroud is fundamentally based on the same tunneling technology as ngrok and srvdir. A shroud client establishes a persistent connection to a public proxy over Tor. That connection is multiplexed to transmit each incoming connection as a separate stream over the persistent connection. The public proxies determine which backend shroud service to route traffic to by inspecting the hostname present in the TLS SNI extension for each incoming TLS connection.

For understanding more about how the tunneling works, you can look at the documentation of the libraries [go-tunnel](https://github.com/inconshreveable/go-tunnel) and [muxado](https://github.com/inconshreveable/muxado) which provide the primitives that enable this behavior.

For those of you who are familiar with ngrok and other localhost-tunneling services which allow you to receive connections to services running on machines behind NATs, you can intuitively understand how shroud works by just thinking of Tor as a NAT. Tor is a large, decentralized, encrypted, onion-routed NAT. Tor clients can establish connections out to the public internet, but no connections can be made into the network. shroud circumvents this limitation by making a single connection out of the Tor network and relaying all inbound traffic over that connection.

## Development

#### How do I build the shroud client?

    make client

If you want to build shroud with Tor compiled in, you'll want to run:

    make client TOR=1

This will compile Tor and all of its dependencies into the shroud client binary. This is known to work on Linux and OS X. Compiling Tor in does not yet work on Windows.

I need to modify the code to use "snakeoil" TLS certificates and keys when compiled in debug mode to make development easier. Stay tuned.

## FAQ

#### Do I need to install Tor?

**No.** Shroud is distributed as a single binary with all of its dependencies compiled in, including Tor (on Linux and OS X), so you don't need to worry about installing anything else. If you've already got Tor installed, though, don't worry, that's OK too, see the next question.

#### I already have Tor installed, can I tell shroud to use that instead of starting its own?

**Yes.** Just pass the -torAddr switch to shroud and it will skip starting Tor and instead use the one at the provided address. By default, Tor's SOCKS5 proxy runs on port 9050, so you'll probably want the switch to look like this:

    ./shroud -torAddr="127.0.0.1:9050" example.com 5555

#### Can I run non-HTTP services over shroud?

**Yes.** You can run any TCP service over shroud so long as you access it via TLS and any clients you support connecting to your shrouded services support setting the TLS SNI extension.

#### Can I run my own shroud public proxies instead of using the ones provided by shroud.io?

**Yes.** The source code for the shroud public proxy is under src/server. You can build a public shroud proxy with the command:

    make server

Even though building the server is easy, getting everything to work is a bit more challenging because it requires that the client and server can talk to each other with their own TLS keys that you will need to generate and compile in to a custom build of the client. I'll add better docs on how to do this, but for now, just contact me if you're interested.

#### Can I run shroud on Windows?

**Yes.** Shroud will work properly on Windows, but I haven't yet done the work to build a version of shroud for Windows with Tor compiled in. If you want to run shroud on Windows, you'll need to first download and run Tor and then point shroud at your running Tor with the -torAddr switch.

#### So can't these public proxies just man-in-the-middle connections to shrouded services?

**No.** All traffic to shrouded services are encrypted with TLS *with keys to the domain that only the service provider controls*. This means that the shroud public proxies can not inspect or modify traffic to shrouded services.

NB: A misbehaving shroud proxy could drop traffic to a shrouded service.

NB: Shrouded services are still at the mercy of the broken CA web of trust. If clients do not practice certificate pinning, a misbehaving shroud proxy controlled by someone who can issue certificates trusted by browsers could MITM traffic.

#### You know about Tor2Web right? Isn't this the same thing?
**It's similar.** Shroud differs from Tor2Web in a few important and subtle ways:

1. Shrouded services are accessed by going directly to their public domain name unlike tor2web addresses which are subdomains
1. Shroud proxies do not and cannot inspect any of the traffic that they proxy to shrouded services because all proxied traffic is TLS encrypted with keys that the proxy does not control.
1. Shrouded services do not use the Tor hidden service protocol and have no corresponding .onion address.
1. Shroud proxies only allow connections to the shrouded services that are connected to them, they provide no reachability to traditional Tor hidden services.
1. Because the shroud protocol runs *over* the traditional Tor network, it will also work on top of any other anonymity network with little or no modification.

## License
Apache
