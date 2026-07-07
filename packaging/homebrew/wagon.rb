class Wagon < Formula
  desc "Terminal file manager for rclone"
  homepage "https://github.com/OverStackedLab/wagon"
  url "https://github.com/OverStackedLab/wagon/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "REPLACE_WITH_RELEASE_TARBALL_SHA256"
  license "MIT"

  depends_on "go" => :build
  depends_on "rclone"

  def install
    ldflags = "-X github.com/OverStackedLab/wagon/internal/cli.version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags, output: bin/"wagon"), "./cmd/wagon"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/wagon version")
    assert_match "Wagon is a terminal file manager", shell_output("#{bin}/wagon --help")
  end
end
