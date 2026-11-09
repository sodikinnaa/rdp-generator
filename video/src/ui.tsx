import React from "react";
import {
  AbsoluteFill,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { colors, fonts } from "./theme";

export const Background: React.FC = () => {
  const frame = useCurrentFrame();
  const drift = interpolate(frame, [0, 600], [0, 120]);
  return (
    <AbsoluteFill
      style={{
        background: `radial-gradient(1200px 800px at ${20 + drift / 6}% 30%, #1b2a5c 0%, ${colors.bg} 55%),
                     radial-gradient(900px 700px at ${85 - drift / 8}% 80%, #0e3a4a 0%, transparent 60%)`,
        backgroundColor: colors.bg,
      }}
    >
      <AbsoluteFill
        style={{
          backgroundImage:
            "linear-gradient(rgba(255,255,255,0.04) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.04) 1px, transparent 1px)",
          backgroundSize: "64px 64px",
          transform: `translate(${-drift / 3}px, ${-drift / 5}px)`,
        }}
      />
    </AbsoluteFill>
  );
};

/** Fade+slide-up entrance for any child. */
export const Reveal: React.FC<{
  delay?: number;
  children: React.ReactNode;
  style?: React.CSSProperties;
  from?: number;
}> = ({ delay = 0, children, style, from = 40 }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const p = spring({ frame: frame - delay, fps, config: { damping: 200 } });
  return (
    <div
      style={{
        opacity: p,
        transform: `translateY(${(1 - p) * from}px)`,
        ...style,
      }}
    >
      {children}
    </div>
  );
};

/** Fades the whole scene in and out at its edges. */
export const SceneFade: React.FC<{
  durationInFrames: number;
  children: React.ReactNode;
}> = ({ durationInFrames, children }) => {
  const frame = useCurrentFrame();
  const opacity = interpolate(
    frame,
    [0, 12, durationInFrames - 12, durationInFrames],
    [0, 1, 1, 0],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
  );
  return <AbsoluteFill style={{ opacity }}>{children}</AbsoluteFill>;
};

export const Title: React.FC<{ children: React.ReactNode; size?: number }> = ({
  children,
  size = 88,
}) => (
  <h1
    style={{
      fontFamily: fonts.sans,
      fontSize: size,
      fontWeight: 800,
      letterSpacing: -2,
      color: colors.text,
      margin: 0,
      lineHeight: 1.05,
    }}
  >
    {children}
  </h1>
);

export const Subtitle: React.FC<{ children: React.ReactNode; size?: number }> = ({
  children,
  size = 40,
}) => (
  <p
    style={{
      fontFamily: fonts.sans,
      fontSize: size,
      color: colors.muted,
      margin: 0,
      lineHeight: 1.35,
    }}
  >
    {children}
  </p>
);

export const Tag: React.FC<{ children: React.ReactNode; color?: string }> = ({
  children,
  color = colors.accent,
}) => (
  <span
    style={{
      fontFamily: fonts.mono,
      fontSize: 26,
      color,
      border: `2px solid ${color}`,
      borderRadius: 999,
      padding: "6px 20px",
      background: "rgba(0,0,0,0.25)",
    }}
  >
    {children}
  </span>
);

export const Panel: React.FC<{
  children: React.ReactNode;
  style?: React.CSSProperties;
  glow?: string;
}> = ({ children, style, glow }) => (
  <div
    style={{
      background: colors.panel,
      border: `2px solid ${colors.border}`,
      borderRadius: 24,
      padding: 36,
      boxShadow: glow ? `0 0 60px ${glow}33` : undefined,
      backdropFilter: "blur(6px)",
      ...style,
    }}
  >
    {children}
  </div>
);

/** Monospace terminal-style box that types out lines. */
export const Terminal: React.FC<{
  lines: string[];
  startFrame?: number;
  charsPerFrame?: number;
  width?: number;
}> = ({ lines, startFrame = 0, charsPerFrame = 1.5, width = 1100 }) => {
  const frame = useCurrentFrame();
  const budget = Math.max(0, (frame - startFrame) * charsPerFrame);
  let remaining = budget;
  const rendered = lines.map((line) => {
    const take = Math.min(line.length, Math.max(0, Math.floor(remaining)));
    remaining -= line.length + 8; // small pause between lines
    return line.slice(0, take);
  });
  const cursorOn = Math.floor(frame / 15) % 2 === 0;
  // Cursor sits only on the last line that has any text so far.
  let cursorLine = -1;
  rendered.forEach((l, i) => {
    if (l.length > 0) cursorLine = i;
  });
  return (
    <div
      style={{
        width,
        background: "#05070f",
        border: `2px solid ${colors.border}`,
        borderRadius: 20,
        overflow: "hidden",
        fontFamily: fonts.mono,
        fontSize: 30,
        color: colors.text,
        boxShadow: "0 30px 80px rgba(0,0,0,0.5)",
      }}
    >
      <div
        style={{
          display: "flex",
          gap: 12,
          padding: "16px 22px",
          background: "rgba(255,255,255,0.04)",
          borderBottom: `1px solid ${colors.border}`,
        }}
      >
        {[colors.red, colors.amber, colors.green].map((c) => (
          <div
            key={c}
            style={{ width: 16, height: 16, borderRadius: 8, background: c }}
          />
        ))}
      </div>
      <div style={{ padding: "28px 32px", lineHeight: 1.7, minHeight: 240 }}>
        {rendered.map((l, i) => {
          const isLast = i === cursorLine;
          const isCmd = lines[i].startsWith("$");
          return (
            <div key={i} style={{ whiteSpace: "pre" }}>
              <span style={{ color: isCmd ? colors.accent2 : colors.muted }}>
                {l}
              </span>
              {isLast && l.length > 0 && cursorOn ? (
                <span style={{ color: colors.accent2 }}>▌</span>
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
};
