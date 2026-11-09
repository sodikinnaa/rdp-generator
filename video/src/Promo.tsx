import React from "react";
import { AbsoluteFill, Audio, Sequence, staticFile } from "remotion";
import { Background, SceneFade } from "./ui";
import {
  ArchitectureScene,
  FeaturesScene,
  IntroScene,
  OutroScene,
  ProblemScene,
  QuickstartScene,
} from "./scenes";

export const FPS = 30;

const scenes: { component: React.FC; seconds: number }[] = [
  // Timed to the paragraph pauses in public/voiceover.wav (52.4 s).
  { component: IntroScene, seconds: 5.7 },
  { component: ProblemScene, seconds: 6.0 },
  { component: ArchitectureScene, seconds: 11.1 },
  { component: FeaturesScene, seconds: 9.95 },
  { component: QuickstartScene, seconds: 9.25 },
  { component: OutroScene, seconds: 11.0 },
];

export const PROMO_DURATION = scenes.reduce(
  (acc, s) => acc + Math.round(s.seconds * FPS),
  0
);

export const Promo: React.FC = () => {
  let from = 0;
  return (
    <AbsoluteFill>
      <Audio src={staticFile("voiceover.wav")} />
      <Background />
      {scenes.map(({ component: Scene, seconds }, i) => {
        const len = Math.round(seconds * FPS);
        const start = from;
        from += len;
        return (
          <Sequence key={i} from={start} durationInFrames={len}>
            <SceneFade durationInFrames={len}>
              <Scene />
            </SceneFade>
          </Sequence>
        );
      })}
    </AbsoluteFill>
  );
};
