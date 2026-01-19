class Gn < Formula
  desc "Send nudge messages to Claude agents in tmux windows"
  homepage "https://github.com/nmelo/gasnudge"
  url "https://github.com/nmelo/gasnudge/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "PLACEHOLDER"
  license "MIT"
  head "https://github.com/nmelo/gasnudge.git", branch: "main"

  depends_on "go" => :build
  depends_on "tmux"

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w", output: bin/"gn")
  end

  test do
    assert_match "gn", shell_output("#{bin}/gn --help")
  end
end
