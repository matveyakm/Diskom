import React, {SVGProps, useEffect, useState} from "react";

interface DynamicIconProps extends SVGProps<SVGSVGElement> {
    iconFromBackend: string;
}

export const DynamicIcon = ({iconFromBackend, ...props}: DynamicIconProps) => {
    const [IconComponent, setIconComponent] = useState<React.FC<SVGProps<SVGSVGElement>> | null>(null);

    useEffect(() => {
        const cleanName = iconFromBackend.replace(/\.png$/, "");
        import(`../../assets/icons/${cleanName}.svg?react`)
            .then((module) => {
                setIconComponent(() => module.default);
            })
            .catch((err) => {
                console.error(`Не удалось загрузить SVG иконку для: ${cleanName}`, err);
            });
    }, [iconFromBackend]);

    if (!IconComponent) {
        return <span style={{width: 24, height: 24, display: "inline-block"}}/>;
    }

    return <IconComponent {...props} />;
};
