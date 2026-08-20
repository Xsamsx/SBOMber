import React from "react";

type BadgeProps = {
    label: string;
};

export function Badge(props: BadgeProps) {
    return <span>{props.label}</span>;
}
