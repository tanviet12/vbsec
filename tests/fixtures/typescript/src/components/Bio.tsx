import React from "react";

type Props = { name: string; bio: string };

export function Bio({ name, bio }: Props) {
  return (
    <section className="bio">
      <h2>{name}</h2>
      <p>{bio}</p>
    </section>
  );
}
