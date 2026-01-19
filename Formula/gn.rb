class Gn < Formula
  desc "Send nudge messages to Claude agents in tmux windows"
  homepage "https://github.com/nmelo/gasnudge"
  url "https://github.com/nmelo/gasnudge/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "3691917bf89e32de768e210bf200dff793519e5121961f4f2d7eab59f51eb336"
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
