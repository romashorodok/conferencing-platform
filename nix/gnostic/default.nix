{ lib, buildGoModule, fetchFromGitHub }:

# git ls-remote https://github.com/google/gnostic main

buildGoModule rec {
  name = "gnostic-main";

  src = fetchFromGitHub {
    owner = "google";
    repo = name;
    rev = "ee84fd2a96205f519ad7b86d989673d2ada03a3b";
    sha256 = "Wpe+rK4XMfMZYhR1xTEr0nsEjRGkSDA7aiLeBbGcRpA=";
  };

  vendorHash = "sha256-Wyv5czvD3IwE236vlAdq8I/DnhPXxdbwZtUhun+97x4=";

  doCheck = false;
}
