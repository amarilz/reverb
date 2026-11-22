pub fn normalize(bytes: &[u8]) -> String {
    let decoded = String::from_utf8(bytes.to_vec()).unwrap_or_else(|_| {
        // The Go implementation tries Windows-1252 first, which accepts every byte.
        const WINDOWS_1252: [char; 32] = [
            '€', '\u{81}', '‚', 'ƒ', '„', '…', '†', '‡', 'ˆ', '‰', 'Š', '‹', 'Œ', '\u{8d}', 'Ž',
            '\u{8f}', '\u{90}', '‘', '’', '“', '”', '•', '–', '—', '˜', '™', 'š', '›', 'œ',
            '\u{9d}', 'ž', 'Ÿ',
        ];
        bytes
            .iter()
            .map(|&byte| match byte {
                0x80..=0x9f => WINDOWS_1252[(byte - 0x80) as usize],
                _ => char::from(byte),
            })
            .collect()
    });
    clean_links(&decoded.replace("\r\n", "\n").replace('\r', "\n"))
}

fn clean_links(mut text: &str) -> String {
    let mut output = String::with_capacity(text.len());
    while let Some(open) = text.find('[') {
        output.push_str(&text[..open]);
        text = &text[open + 1..];
        if let Some(close) = text.find(']')
            && let Some(url) = text[close + 1..].strip_prefix('(')
            && let Some(end) = url.find(')')
        {
            output.push_str(&text[..close]);
            text = &url[end + 1..];
        } else {
            output.push('[');
        }
    }
    output.push_str(text);
    output.trim().to_owned()
}

pub fn strip_code_blocks(text: &str) -> String {
    let mut inside = false;
    text.split('\n')
        .filter(|line| {
            if line.trim().starts_with("```") {
                inside = !inside;
                return false;
            }
            !inside
        })
        .collect::<Vec<_>>()
        .join("\n")
        .trim()
        .to_owned()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn normalizes_clipboard_text() {
        for (input, expected) in [
            ("", ""),
            ("  Héllo 🌍  ", "Héllo 🌍"),
            ("a\r\nb\rc", "a\nb\nc"),
            (
                "read [one](https://one.com) and [two]()",
                "read one and two",
            ),
            ("link [](https://example.com) empty", "link  empty"),
            ("test [incomplete link", "test [incomplete link"),
            ("test [label] without url", "test [label] without url"),
            (
                "test [label](https://example.com",
                "test [label](https://example.com",
            ),
        ] {
            assert_eq!(normalize(input.as_bytes()), expected);
        }
        assert_eq!(normalize(b"caf\xe9"), "café");
    }

    #[test]
    fn strips_multiple_fenced_blocks_and_keeps_inline_code() {
        assert_eq!(
            strip_code_blocks(
                "Before\n\n```\ncode\n```\n\nMiddle\n\n```js\ncode\n```\n\nAfter `inline`"
            ),
            "Before\n\n\nMiddle\n\n\nAfter `inline`"
        );
    }
}
