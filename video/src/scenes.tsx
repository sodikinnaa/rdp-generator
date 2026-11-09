import React from "react";
import {
  AbsoluteFill,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { colors, fonts } from "./theme";
import { Panel, Reveal, Subtitle, Tag, Terminal, Title } from "./ui";

const Center: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <AbsoluteFill
    style={{
      justifyContent: "center",
      alignItems: "center",
      padding: 120,
    }}
  >
    {children}
  </AbsoluteFill>
);

// ---------------------------------------------------------------- 1. Intro
export const IntroScene: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const logo = spring({ frame, fps, config: { damping: 12, stiffness: 120 } });
  return (
    <Center>
      <div style={{ textAlign: "center" }}>
        <div
          style={{
            display: "inline-flex",
            width: 170,
            height: 170,
            borderRadius: 44,
            background: `linear-gradient(135deg, ${colors.accent}, ${colors.accent2})`,
            justifyContent: "center",
            alignItems: "center",
            transform: `scale(${logo}) rotate(${(1 - logo) * -20}deg)`,
            boxShadow: `0 30px 80px ${colors.accent}55`,
            marginBottom: 48,
            fontFamily: fonts.mono,
            fontSize: 72,
            fontWeight: 800,
            color: "#fff",
          }}
        >
          ⌘
        </div>
        <Reveal delay={10}>
          <Title size={120}>RDP Generator</Title>
        </Reveal>
        <Reveal delay={22}>
          <Subtitle size={46}>
            Windows Server on GitHub Codespaces, over Tailscale.
          </Subtitle>
        </Reveal>
        <Reveal delay={34} style={{ marginTop: 40 }}>
          <div style={{ display: "flex", gap: 20, justifyContent: "center" }}>
            <Tag>Go single binary</Tag>
            <Tag color={colors.accent2}>Open source · MIT</Tag>
            <Tag color={colors.green}>Zero public ports</Tag>
          </div>
        </Reveal>
      </div>
    </Center>
  );
};

// ------------------------------------------------------------- 2. Problem
export const ProblemScene: React.FC = () => {
  const items = [
    "Need a throwaway Windows box for testing?",
    "Don't want to rent a VPS or open RDP to the internet?",
    "Want it in one command from your laptop?",
  ];
  return (
    <Center>
      <div style={{ maxWidth: 1500 }}>
        <Reveal>
          <Tag color={colors.amber}>The problem</Tag>
        </Reveal>
        <div style={{ height: 36 }} />
        {items.map((t, i) => (
          <Reveal key={t} delay={10 + i * 14}>
            <Title size={74}>{t}</Title>
            <div style={{ height: 28 }} />
          </Reveal>
        ))}
      </div>
    </Center>
  );
};

// -------------------------------------------------------- 3. Architecture
const Box: React.FC<{
  title: string;
  lines: string[];
  color: string;
  delay: number;
  width?: number;
}> = ({ title, lines, color, delay, width = 460 }) => (
  <Reveal delay={delay} from={30}>
    <Panel glow={color} style={{ width }}>
      <div
        style={{
          fontFamily: fonts.sans,
          fontSize: 36,
          fontWeight: 700,
          color,
          marginBottom: 14,
        }}
      >
        {title}
      </div>
      {lines.map((l) => (
        <div
          key={l}
          style={{
            fontFamily: fonts.mono,
            fontSize: 26,
            color: colors.muted,
            lineHeight: 1.6,
          }}
        >
          {l}
        </div>
      ))}
    </Panel>
  </Reveal>
);

const Arrow: React.FC<{ delay: number; label: string }> = ({ delay, label }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const p = spring({ frame: frame - delay, fps, config: { damping: 200 } });
  return (
    <div
      style={{
        width: 170,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        gap: 10,
        opacity: p,
      }}
    >
      <div
        style={{
          fontFamily: fonts.mono,
          fontSize: 22,
          color: colors.accent2,
          textAlign: "center",
        }}
      >
        {label}
      </div>
      <div
        style={{
          width: 150 * p,
          height: 4,
          background: `linear-gradient(90deg, ${colors.accent}, ${colors.accent2})`,
          borderRadius: 4,
          position: "relative",
        }}
      >
        <div
          style={{
            position: "absolute",
            right: -8,
            top: -10,
            width: 0,
            height: 0,
            borderTop: "12px solid transparent",
            borderBottom: "12px solid transparent",
            borderLeft: `16px solid ${colors.accent2}`,
          }}
        />
      </div>
    </div>
  );
};

export const ArchitectureScene: React.FC = () => {
  return (
    <Center>
      <div style={{ width: "100%" }}>
        <Reveal>
          <Tag>How it works</Tag>
        </Reveal>
        <div style={{ height: 50 }} />
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            gap: 10,
          }}
        >
          <Box
            delay={8}
            color={colors.accent}
            title="Your laptop"
            lines={[
              "rdp-generator",
              "http://127.0.0.1:8787",
              "Encrypts secrets (NaCl)",
            ]}
          />
          <Arrow delay={22} label="Codespaces API" />
          <Box
            delay={30}
            color={colors.accent2}
            title="GitHub Codespace"
            lines={[
              "dockur/windows (QEMU+KVM)",
              "Windows Server 2022",
              "tailscaled  → 100.x.x.x",
            ]}
            width={520}
          />
          <Arrow delay={46} label="WireGuard mesh" />
          <Box
            delay={54}
            color={colors.green}
            title="Any device on tailnet"
            lines={["RDP   100.x.x.x:3389", "noVNC 100.x.x.x:8006", "No public ports"]}
          />
        </div>
      </div>
    </Center>
  );
};

// ------------------------------------------------------------ 4. Features
export const FeaturesScene: React.FC = () => {
  const feats = [
    ["🧱", "Single binary", "Pure Go + embedded dashboard. No runtimes."],
    ["🔒", "Zero exposed ports", "RDP & noVNC bound to the Tailscale IP only."],
    ["⚡", "KVM accelerated", "Uses /dev/kvm on Codespaces hosts."],
    ["🔐", "Sealed-box secrets", "Keys encrypted client-side before GitHub."],
    ["💾", "Persistent disk", "Windows survives Codespace stop/start."],
    ["🖥️", "Cross-platform", "Windows, Linux and macOS controller."],
  ];
  return (
    <Center>
      <div style={{ width: "100%" }}>
        <Reveal>
          <Tag color={colors.green}>Features</Tag>
        </Reveal>
        <div style={{ height: 44 }} />
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(3, 1fr)",
            gap: 28,
          }}
        >
          {feats.map(([icon, h, d], i) => (
            <Reveal key={h} delay={8 + i * 7}>
              <Panel style={{ height: 230 }}>
                <div style={{ fontSize: 52, marginBottom: 10 }}>{icon}</div>
                <div
                  style={{
                    fontFamily: fonts.sans,
                    fontSize: 36,
                    fontWeight: 700,
                    color: colors.text,
                    marginBottom: 8,
                  }}
                >
                  {h}
                </div>
                <div
                  style={{
                    fontFamily: fonts.sans,
                    fontSize: 26,
                    color: colors.muted,
                    lineHeight: 1.4,
                  }}
                >
                  {d}
                </div>
              </Panel>
            </Reveal>
          ))}
        </div>
      </div>
    </Center>
  );
};

// ---------------------------------------------------------- 5. Quickstart
export const QuickstartScene: React.FC = () => {
  const frame = useCurrentFrame();
  const steps = [
    "Enter GitHub token + Tailscale auth key",
    "Pick Windows version & machine type",
    "Click Create — RDP address appears in the dashboard",
  ];
  return (
    <Center>
      <div
        style={{
          display: "flex",
          gap: 60,
          alignItems: "center",
          width: "100%",
        }}
      >
        <div style={{ flex: 1 }}>
          <Reveal>
            <Tag color={colors.accent2}>Quick start</Tag>
          </Reveal>
          <div style={{ height: 30 }} />
          {steps.map((s, i) => {
            const on = frame > [72, 120, 165][i];
            return (
              <Reveal key={s} delay={10 + i * 10}>
                <div
                  style={{
                    display: "flex",
                    gap: 22,
                    alignItems: "center",
                    marginBottom: 26,
                  }}
                >
                  <div
                    style={{
                      width: 56,
                      height: 56,
                      borderRadius: 28,
                      background: on ? colors.green : "rgba(255,255,255,0.08)",
                      color: on ? "#06251a" : colors.muted,
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      fontFamily: fonts.mono,
                      fontSize: 28,
                      fontWeight: 800,
                      transition: "background 0.2s",
                    }}
                  >
                    {on ? "✓" : i + 1}
                  </div>
                  <div
                    style={{
                      fontFamily: fonts.sans,
                      fontSize: 34,
                      color: colors.text,
                    }}
                  >
                    {s}
                  </div>
                </div>
              </Reveal>
            );
          })}
        </div>
        <Reveal delay={6}>
          <Terminal
            width={960}
            startFrame={14}
            charsPerFrame={2}
            lines={[
              "$ go build -ldflags=\"-s -w\" -o rdp-generator .",
              "$ ./rdp-generator -open",
              "Listening on http://127.0.0.1:8787",
              "",
              "$ mstsc /v:100.101.102.103:3389",
              "",
            ]}
          />
        </Reveal>
      </div>
    </Center>
  );
};

// --------------------------------------------------------------- 6. Outro
export const OutroScene: React.FC = () => {
  const frame = useCurrentFrame();
  const pulse = 1 + Math.sin(frame / 10) * 0.015;
  return (
    <Center>
      <div style={{ textAlign: "center" }}>
        <Reveal>
          <Title size={96}>Open source. MIT licensed.</Title>
        </Reveal>
        <div style={{ height: 30 }} />
        <Reveal delay={12}>
          <Subtitle size={42}>Star it, fork it, ship your own Windows lab.</Subtitle>
        </Reveal>
        <div style={{ height: 60 }} />
        <Reveal delay={24}>
          <div
            style={{
              display: "inline-block",
              transform: `scale(${pulse})`,
              fontFamily: fonts.mono,
              fontSize: 44,
              color: "#fff",
              background: `linear-gradient(135deg, ${colors.accent}, ${colors.accent2})`,
              padding: "22px 48px",
              borderRadius: 20,
              boxShadow: `0 30px 80px ${colors.accent}55`,
            }}
          >
            github.com/sodikinnaa/rdp-generator
          </div>
        </Reveal>
      </div>
    </Center>
  );
};

export const sceneOpacity = (frame: number, len: number) =>
  interpolate(frame, [0, 10, len - 10, len], [0, 1, 1, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
