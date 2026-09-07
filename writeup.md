What I Learned Building a Double Ratchet Without Dependencies

A stolen key does not unlock your conversation history.

For this project, I decided to impose an unnecessarily painful constraint on myself: build a Signal-style double ratchet messenger in Go without importing any third-party package.

No golang.org/x/crypto.

No WebSocket library.

No protobuf.

No CLI framework.

No assertion library.

Just Go's standard library.

Because apparently writing cryptographic messaging software wasn't difficult enough.

The result is MiniRatchet, a zero-dependency implementation of the core ideas behind the Double Ratchet Algorithm. It demonstrates per-message key evolution, authenticated encryption, X25519-based DH ratcheting, forward secrecy, and post-compromise recovery. The repository's go.mod contains no require block, and the project builds using only the Go toolchain.

The most valuable part of the project, however, wasn't the final messenger.

It was discovering what the packages I normally would have installed were actually doing for me.

The constraint

The project started with a simple question:

How much of a modern secure messaging system can actually be built using only Go's standard library?

The obvious approach would have been to install the usual packages and start assembling them.

For example:

• golang.org/x/crypto/hkdf for HKDF
• golang.org/x/crypto/curve25519 for X25519
• golang.org/x/crypto/chacha20poly1305 for AEAD
• a WebSocket package for transport
• protobuf for serialization
• Cobra for CLI handling
• Testify for assertions

Instead, every one of those conveniences became a question:

What is this package actually giving me?

That changed the project from "implement a messenger" into "understand the abstraction underneath every dependency."

1. The first surprise: X25519 was already in the standard library

One of the biggest discoveries was crypto/ecdh.

Go's standard library provides X25519 through:

go
ecdh.X25519()

That meant I didn't need to pull in golang.org/x/crypto/curve25519 simply to perform the Diffie-Hellman portion of the ratchet.

The DH ratchet became conceptually straightforward:

1. Generate an ephemeral X25519 key pair.
2. Exchange public keys.
3. Perform the ECDH operation.
4. Feed the shared secret into the root-key derivation.
5. Replace the old DH state.
6. Continue with a new symmetric chain.

The important lesson was that "zero dependency" does not mean "implement cryptography yourself."

It means compose trusted primitives that the platform already provides.

That distinction matters enormously in cryptographic software.

2. HKDF looked like a package dependency until I looked underneath it

The next problem was key derivation.

Normally, I would reach for:

text
golang.org/x/crypto/hkdf

But HKDF is built from HMAC.

And Go already provides:

text
crypto/hmac
crypto/sha256

So instead of treating HKDF as a magical primitive, I implemented the extract-and-expand construction using HMAC-SHA256.

That became one of the most useful moments in the project.

The dependency had been hiding a relatively understandable construction.

The important part wasn't saving an import.

It was understanding the relationship:

text
shared secret
|
v
HMAC-SHA256
|
v
pseudorandom key material
|
v
expand
|
v
root / chain material

The same idea also powers the symmetric ratchet.

3. The symmetric ratchet is where the security model becomes tangible

The symmetric ratchet was probably the clearest demonstration of why the Double Ratchet works.

Instead of encrypting every message with one long-lived key, the chain advances:

text
Chain Key N
|
+----> Message Key N
|
v
Chain Key N+1
|
+----> Message Key N+1
|
v
Chain Key N+2

The message key is derived and then the chain advances.

The old chain state is discarded.

That creates an important property:

A stolen message key does not automatically reveal the rest of the conversation.

If an attacker obtains message key K₅, they can potentially decrypt message 5.

They cannot simply run the chain backwards to obtain K₄.

And because the chain has already advanced, K₅ isn't reused to encrypt message 6.

This makes the security property visible rather than theoretical.

MiniRatchet includes a steal-key demonstration specifically for this reason. An attacker captures one message key, attempts to decrypt an earlier and later message, and both attempts fail authentication.

4. Then came the more interesting compromise: stealing a chain key

A message key compromise is useful to demonstrate, but it isn't the worst case.

What happens if the attacker gets the chain key?

That's much more serious.

A chain key allows an attacker to derive future message keys within that ratchet epoch.

So the symmetric ratchet alone isn't enough.

This is where the DH ratchet becomes important.

A new DH exchange creates fresh shared secret material and feeds it into the root-key derivation.

Conceptually:

text
Compromised Chain Key
|
X
|
|  DH ratchet
|
v
New Root Key
|
v
New Chain Key

The attacker's old chain key has no path into the new chain.

This is the "self-healing" property I wanted the demo to make obvious.

MiniRatchet's compromise mode simulates exactly this situation: a chain key is leaked, a new DH epoch occurs, and the compromised material becomes disconnected from the newly derived root state.

5. AES-GCM replaced another dependency

For authenticated encryption, I used:

text
crypto/aes
crypto/cipher
crypto/rand

to construct AES-256-GCM.

That provides both confidentiality and authentication.

The message flow becomes:

text
plaintext
|
v
Message Key
|
v
AES-256-GCM
|
+---- random nonce
|
+---- associated data
|
v
ciphertext + authentication tag

The associated data is particularly important because encryption isn't just about hiding the message body.

The protocol also needs to authenticate relevant metadata.

This was another place where using the standard library forced me to understand the API rather than treating an external package as a black box.

6. I also discovered that "transport" is mostly an agreement

I initially thought a messaging demo would need something more sophisticated than TCP.

It didn't.

For the purposes of this project, I only needed a reliable byte stream and a way to determine where one message ended and another began.

So instead of WebSockets and protobuf, I used:

text
net
encoding/binary

with a simple length-prefixed format:

text
[4-byte length][payload]

The receiver reads the length first, then exactly that many bytes.

It isn't glamorous.

It also works.

This was one of the more useful engineering lessons from the entire project:

Sometimes a dependency is solving a complexity problem that you don't actually have.

7. Even the CLI became a dependency audit

I used Go's flag package instead of Cobra.

That gave me commands such as:

bash
./miniratchet --demo steal-key
./miniratchet --demo compromise
./miniratchet --demo two-panel
./miniratchet --demo destroy

Again, the point wasn't that flag is better than Cobra.

The point was discovering that for a small CLI, I didn't need a framework at all.

The same happened with testing.

Instead of Testify:

go
testing

was enough.

Instead of a terminal UI framework and color library, the demo uses fmt, strings, Unicode symbols, and box-drawing characters.

The dependency substitution table in the repository ended up becoming one of the most useful artifacts from the project because it shows exactly what each external package would normally have provided.

8. The part the documentation made look easier than it was

The hardest part wasn't calling cryptographic APIs.

It was maintaining state correctly.

A ratchet isn't just:

text
encrypt(message)
decrypt(message)

It is a state machine.

There are:

• root keys
• sending chain keys
• receiving chain keys
• DH key pairs
• remote DH public keys
• message keys
• ratchet epochs
• state transitions
• key destruction

A single mistake in when state advances can produce a system that appears to work while violating the security property it is supposed to demonstrate.

That changed how I approached testing.

I stopped asking only:

"Can Alice encrypt something that Bob can decrypt?"

and started asking:

"What happens if the attacker obtains this exact piece of state at this exact moment?"

That produced much better tests.

9. The demos became security tests

The final project has four deliberately adversarial demonstrations.

Steal a message key

The attacker obtains one message key.

They can decrypt that message.

They cannot decrypt earlier or later messages.

Compromise a chain key

The attacker obtains a chain key.

They can derive future keys in the current epoch.

Then the DH ratchet runs.

The compromised chain is abandoned.

Two-panel view

One side sees:

text
Hello Bob, this message is secret.

The attacker sees something resembling:

text
8f4a91c7d2...

This is intentionally simple, because cryptographic security is much easier to understand when you can see the two perspectives simultaneously.

Destroy

Finally, the system explicitly wipes key material from its buffers.

The demonstration ends with zeroed memory.

The message is:

There is nothing to recover.

The repository documents all four modes as part of the five-minute demonstration flow.

10. What I deliberately did not solve

A security project becomes more credible when it explains its boundaries.

MiniRatchet is not Signal.

It implements core Double Ratchet concepts for demonstration and experimentation.

It does not attempt to provide the complete security architecture of a production messenger.

For example, it does not currently provide:

Identity authentication

The initial key exchange does not establish that Alice is actually talking to Bob.

An active MITM could interfere with an unauthenticated initial exchange.

Endpoint security

If an attacker completely compromises a device and can inspect its memory, the ratchet cannot magically protect secrets that are already available to the endpoint.

The ratchet protects the communication channel, not a compromised machine.

Full out-of-order message handling

The current symmetric ratchet assumes ordered progression.

A production-grade Double Ratchet implementation needs message numbers and skipped-message-key handling to deal with delayed or reordered messages.

Side-channel resistance

The implementation relies on the guarantees provided by Go's cryptographic primitives. It is not a claim of comprehensive side-channel resistance.

These limitations aren't bugs that I am hiding.

They define the boundary of what this project is demonstrating.

11. The real lesson: zero dependencies changed how I think

The biggest result of this project wasn't a smaller go.mod.

It was learning to ask:

What does this dependency actually contain?

When you install:

text
crypto library
networking library
serialization library
CLI framework
testing framework

it's easy to think of them as indivisible pieces.

They aren't.

Sometimes the standard library already contains the primitive you need.

Sometimes the dependency is only providing a thin wrapper.

Sometimes you don't need the abstraction at all.

And sometimes the dependency really is valuable because it solves a difficult problem that you should not casually reimplement.

That last point is important for cryptography.

Zero dependency should not become "I wrote my own AES."

That would be spectacularly irresponsible.

The goal is to minimize unnecessary dependencies while relying on mature, standard cryptographic primitives.

Go's standard library already provides X25519, AES-GCM, HMAC-SHA256, and OS-backed cryptographic randomness, which made that approach practical for this project.

12. What I would do next

If I continued MiniRatchet beyond the hackathon, the first improvements would be protocol-level rather than cosmetic:

1. Add authenticated identity keys and an explicit authentication mechanism.
2. Implement message numbers and skipped-message-key storage.
3. Add stronger protocol serialization and versioning.
4. Expand interoperability tests between independent implementations.
5. Add more adversarial and state-transition tests.
6. Perform a proper security review before treating it as anything other than an educational implementation.

The interesting part is that none of those problems can be solved simply by adding another package.

That was probably the most useful thing the zero-dependency constraint taught me.

Conclusion

I started this project expecting "zero dependencies" to mean fewer lines in go.mod.

It turned out to mean something more useful:

I had to understand what every layer was actually doing.

Instead of importing HKDF, I had to understand HMAC-based extraction and expansion.

Instead of importing a Curve25519 package, I discovered X25519 support in crypto/ecdh.

Instead of adding a messaging framework, I implemented a small framing protocol.

Instead of hiding the ratchet behind a library, I had to reason about how individual keys evolve, disappear, and become useless after compromise.

The final result is a small Go program that demonstrates a surprisingly large amount of modern cryptographic protocol design using only the standard library.

And the best outcome wasn't proving that third-party packages are unnecessary.

It was learning when they are necessary, when they aren't, and what they're actually doing when you use them.

That is something I wouldn't have learned by simply running:

bash
go get ...

Humanity has apparently invented package managers specifically so we can avoid finding out what our software does.
