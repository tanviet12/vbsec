import React from "react";

type Props = { author: string; body: string };

export function Comment({ author, body }: Props) {
  return (
    <div className="comment">
      <strong>{author}</strong>
      <div dangerouslySetInnerHTML={{ __html: body }} />
    </div>
  );
}
