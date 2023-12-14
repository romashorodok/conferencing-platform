{ pkgs ? import <nixpkgs> { } }:

let
  gnostic-main = pkgs.callPackage ./nix/gnostic { };
  protoc-gen-fetch-types-main = pkgs.callPackage ./nix/protoc-gen-fetch-types { };
in
pkgs.mkShell {
  buildInputs = with pkgs; [
    gnostic-main
    protoc-gen-fetch-types-main
  ];
}
