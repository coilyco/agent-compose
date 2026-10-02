#!/bin/sh
# Render the brew formula and scoop manifest from dist/ binaries; requires
# an exact release tag so URLs, stamps, and hashes agree.
set -e
VERSION="$(git describe --tags --exact-match)"
BARE="${VERSION#v}"
BASE="https://forgejo.coilysiren.me/coilyco-flight-deck/agent-compose/releases/download/${VERSION}"

sha() { shasum -a 256 "$1" | cut -d' ' -f1; }
DARWIN_ARM64="$(sha dist/agent-compose-darwin-arm64)"
LINUX_AMD64="$(sha dist/agent-compose-linux-amd64)"
LINUX_ARM64="$(sha dist/agent-compose-linux-arm64)"
WINDOWS_AMD64="$(sha dist/agent-compose-windows-amd64.exe)"
ROSTER="$(sha dist/agent-compose-roster.tar.gz)"
BUNDLES="$(sha dist/agent-compose-bundles.tar.gz)"

cat > dist/agent-compose.rb <<EOF
require_relative "../lib/tailnet_download_strategy"

class AgentCompose < Formula
  desc "Core Roster context composition for native agent harnesses"
  homepage "https://forgejo.coilysiren.me/coilyco-flight-deck/agent-compose"
  version "${BARE}"
  license "MIT"

  # The seed roster installs into the prefix. acompose prefers an editable
  # roster in the state directory, so an upgrade never overwrites one.
  resource "roster" do
    url "${BASE}/agent-compose-roster.tar.gz", using: TailnetCurlDownloadStrategy
    sha256 "${ROSTER}"
  end

  # housecast composes at build time and never runs here, so the composed set
  # ships rather than the data behind it.
  resource "bundles" do
    url "${BASE}/agent-compose-bundles.tar.gz", using: TailnetCurlDownloadStrategy
    sha256 "${BUNDLES}"
  end

  on_macos do
    on_arm do
      url "${BASE}/agent-compose-darwin-arm64", using: TailnetCurlDownloadStrategy
      sha256 "${DARWIN_ARM64}"
    end
  end
  on_linux do
    on_intel do
      url "${BASE}/agent-compose-linux-amd64", using: TailnetCurlDownloadStrategy
      sha256 "${LINUX_AMD64}"
    end
    on_arm do
      url "${BASE}/agent-compose-linux-arm64", using: TailnetCurlDownloadStrategy
      sha256 "${LINUX_ARM64}"
    end
  end

  def install
    bin.install Dir["agent-compose-*"].first => "agent-compose"
    bin.install_symlink "agent-compose" => "acompose"
    # Homebrew chdirs into a lone top-level directory before yielding a stage
    # block, so a block cannot name the directory it sits inside: agentic-os#6835.
    resource("roster").stage(share/"agent-compose"/"roster")
    resource("bundles").stage(share/"agent-compose"/"bundles")
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/agent-compose version")
  end
end
EOF

cat > dist/agent-compose.json <<EOF
{
    "version": "${BARE}",
    "description": "Core Roster context composition for native agent harnesses",
    "homepage": "https://forgejo.coilysiren.me/coilyco-flight-deck/agent-compose",
    "license": "MIT",
    "architecture": {
        "64bit": {
            "url": [
                "${BASE}/agent-compose-windows-amd64.exe",
                "${BASE}/agent-compose-roster.tar.gz",
                "${BASE}/agent-compose-bundles.tar.gz"
            ],
            "hash": [
                "${WINDOWS_AMD64}",
                "${ROSTER}",
                "${BUNDLES}"
            ],
            "bin": [
                ["agent-compose-windows-amd64.exe", "agent-compose"],
                ["agent-compose-windows-amd64.exe", "acompose", "compose"]
            ]
        }
    },
    "pre_install": "\$c = New-Object Net.Sockets.TcpClient; try { if (-not \$c.ConnectAsync('forgejo.coilysiren.me', 443).Wait(3000)) { throw 'timeout' } } catch { throw 'agent-compose is only available on the coilyco tailnet. If you''re on the tailnet and have access, connect and retry. If you''re the general public, you don''t have access to this.' } finally { \$c.Dispose() }"
}
EOF

echo "dist/agent-compose.rb"
echo "dist/agent-compose.json"
