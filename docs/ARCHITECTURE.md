## Shroud service architecture

                                TOR NETWORK                                                      
                      +–––––––––––––––––––––––––––––+                                            
                      |                             |                                            
    +––––––––––––––+  | +–+              +–+        |                                            
    | PUBLIC PROXY +––––+ +––––+  +–+    | |        |                                            
    +––––––––––––––+  | +–+    |  | | +–++–+   +–+  |                                            
                      |   +–+ +++ +–+ | |      | |  |     SHROUD CLIENT                          
                      |   | | | |     +–+      +–+  |   +––––––––––––––––+                       
                      |   +–+ +++                   |   |                |                       
                      | +–+    |  +–+               |   | +–––+ +––––––+ |     +––––––––––––––––+
                      | | |    +––+ +–––––––––––––––––––––+TOR+–+TUNNEL+–––––––+SHROUDED SERVICE|
                      | +–+       +–+               |   | +–––+ +––––––+ |     +––––––––––––––––+
                      |                             |   |                |                       
                      +–––––––––––––––––––––––––––––+   +––––––––––––––––+                       


Shroud provides low-latency hidden services with public addresses by establishing secure tunnels through the Tor network to a public proxy.

To expose a new shrouded service, one starts a new shroud client with command line arguments like so:

    ./shroud example.com 127.0.0.1:443

If you are using the shroud.io service, you also created a CNAME record from the domain you wanted to expose (example.com) to a shroud proxy subdomain (example.com.proxy.v1.shroud.io).

#### When the shroud client starts up
1. When the shroud client starts up, it first initiailizes Tor which will create a new circuit through the Tor network to an exit node.
1.  Once that circuit is ready, the tunnel piece of shroud establishes a new long-lived TLS connection to a public proxy over the Tor circuit.
1. Over this TLS connection, the shroud-client will ask the public proxy to forward it any requests it receives that are intended for 'example.com'. This is called *binding* for example.com.
    - The public proxy knows that connections are intended for 'example.com' by inspecting the TLS SNI data or the HTTP Host header.

#### When the public proxy receives a new connection intended for example.com
1. When a new TLS connection comes in, the public proxy examines the SNI data to find which hostname a client is requesting.
    - Do not confuse 'client' here for a shroud client. 'Client' here refers to a public client like a web browser.
1. The shroud public proxy will then consult its internal mapping of hostname -> shroud client tunnel connections.
1. If it finds a shroud client that has bound for example.com, it will open a new logical stream on that connection and begin proxying data to and from the the public TLS connection and the logical stream on the tunnel to the shroud client.
    - All public connections which are intended for example.com are multiplexed as separate streams over a single connection between the shroud client and the shroud public proxy. This stream multiplexing is similar to what's done by SPDY or "ssh -R". You can find the code for it in https://github.com/inconshreveable/muxado
1. When the shroud client receives a new stream from the shroud public proxy, it opens a new connection to 127.0.0.1:443 and begins proxying data to and from that connection and the tunneled stream.

#### Is there more than one public proxy?
Yes.

The diagram above is actually a bit simplified. In reality, when the shroud client starts up, it doesn't start just one tunnel to a public proxy. It actually starts up *multiple tunnels*, one to each public proxy (the default right now is 3 - one in each part of the world). This avoids a single point of failure and can help decrease world-wide latency.

#### How do public clients (like browsers) find the public proxy servers?
Clients like web browsers find the public proxy servers via DNS. If you are using the shroud.io service, you needed to CNAME the domain for your shrouded service to a shroud.io domain. If you chose to run your own public proxy servers, then you could just create A records to each public proxy that you had set up.

#### How does the shroud client know which public proxy servers to create tunnels to?
There are two possibilities for this. The simpler mechanism is that a shroud client can simply just be passed a list of public proxy addresses to connect to at startup time.

The shroud.io service does something a little more clever, though. Instead, when the shroud client starts up, it actually makes a DNS request *through Tor* to look at the SRV records set up for the '_shroud' service on the shrouded domain and uses the returned addresses and ports as the public proxies to connect to.

#### I'm pretty sure Tor doesn't support doing SRV lookups.
And you would be right. There's a new spec proposal out for Tor that supports arbitrary DNS record lookups through Tor, but it's not implemented yet. And even if it was, DNS is easy to compromise at the exit node. So for shroud.io, what actually happens is that the shroud client initiates a TLS connection to the shroud.io "discovery service" which in turn returns the right addresses of the public proxies the client should connect to.

#### You say that I don't need to install Tor first and that shroud starts Tor itself, how does that work?
The shroud client actually has Tor compiled in to its binary and statically links it all together to make deployment easy and reduce the number of runtime dependencies. The magic that makes this happen is predominantly in the Makefile and in the Cgo directives in the client/tor.go file.

#### I didn't quite understand all of the crazy tunneling and multiplexing stuff. Is there more information on that?
The tunneling is done via my go-tunnel library, whose code and docs are here: [https://github.com/inconshreveable/go-tunnel](https://github.com/inconshreveable/go-tunnel)
The code and docs for stream multiplexing (which go-tunnel uses) are from my muxado library here: [https://github.com/inconshreveable/muxado](https://github.com/inconshreveable/muxado)
