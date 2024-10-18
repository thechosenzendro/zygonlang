use std::{fs, vec::Vec};
use std::str::FromStr;

fn main() {
    let src = fs::read_to_string("examples/Main.zg").expect("File not found");
    println!("{src}");

    let tokens = tokenize(&src);
    println!("{:?}", tokens)
}

#[derive(Debug)]
enum Token {
    Identifier(String),
    Colon,
    Number(f64),
}

fn get(text: &String, i: usize) -> char {
    text.chars().nth(i).expect("No character found")
}
// probably change this?
fn tokenize(src: &String) -> Vec<Token>{
    let mut tokens: Vec<Token> = Vec::new();
    let mut i = 0;
    while i < src.len() {
      if get(src, i).is_alphabetic() || get(src, i) == '_' {
          let mut buf = String::new();
          while i < src.len() && (get(src, i).is_ascii_alphanumeric() || get(src, i) == '_') {
              buf.push(get(src, i));
              i += 1;

          }
          tokens.push(Token::Identifier(buf));
      } else if get(src, i) == ':' {
          i += 1;
          tokens.push(Token::Colon);
      } else if get(src, i).is_whitespace() {
          i += 1
      } else if get(src, i).is_numeric() {
          let mut buf = String::new();
          let mut is_float = false;
          while i < src.len() && (get(src, i).is_numeric() || get(src, i) == '.') {
              if get(src, i) == '.' {
                  if is_float {panic!("bad number literal")} else { is_float = true };
              };
              buf.push(get(src, i));
              i += 1;

          }
          tokens.push(Token::Number(f64::from_str(&buf).expect("bad number")));
      } else {
          panic!("Unexpected character {}", get(src, i));
      }
    }
    tokens
}