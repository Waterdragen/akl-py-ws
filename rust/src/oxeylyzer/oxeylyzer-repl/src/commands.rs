use actix::Addr;
use oxeylyzer_ws::sender::Sendable;
use oxeylyzer_ws::websocket::OxeylyzerWs;

#[derive(Debug)]
pub(crate) enum ArgumentType<'a> {
    R(&'a str),
    O(&'a str),
    A(&'a str),
}

impl<'a> ArgumentType<'a> {
    pub(crate) fn is_required(&self) -> bool {
        match *self {
            Self::R(_) => true,
            _ => false,
        }
    }

    pub(crate) fn parse(&self) -> String {
        match *self {
            Self::R(s) => format!("<{s}>"),
            Self::O(s) => format!("[{s}]"),
            Self::A(s) => {
                let first = s.chars().next().unwrap();
                format!("[-{first}/--{s}]")
            }
        }
    }
}

fn usage(command_name: &str, args: &[ArgumentType]) -> String {
    let args_left_right = args
        .into_iter()
        .map(ArgumentType::parse)
        .collect::<Vec<_>>()
        .join(" ");

    format!("USAGE:\n    {command_name} {args_left_right}")
}

pub struct Commands {
    addr: Addr<OxeylyzerWs>,
}
impl Sendable<OxeylyzerWs> for Commands {
    fn addr(&self) -> &Addr<OxeylyzerWs> {
        &self.addr
    }
}
impl Commands {
    pub(crate) fn new(addr: Addr<OxeylyzerWs>) -> Self {
        Commands {addr }
    }

    pub(crate) fn send_help(&self, command_name: &str, about: &str, args: &[ArgumentType]) {
        self.sendln(format!("{about}\n\n{}\n", usage(command_name, args)));
    }

    pub(crate) fn send_error(&self, command_name: &str, args: &[ArgumentType]) {
        let plural = if args.len() > 1 { "s were" } else { " was" };

        let args_top_down = args
            .into_iter()
            .filter(|a| a.is_required())
            .map(ArgumentType::parse)
            .collect::<Vec<_>>()
            .join("\n    ");

        self.sendln(format!(
            concat!(
            "error: The following required argument{} not provided:\n    {}\n\n{}",
            "\n\nFor more information try 'help'"
            ),
            plural,
            args_top_down,
            usage(command_name, args)
        ));
    }

}

