import type { ReactNode } from 'react';
import './Communications.css';
export function CommunicationsHeader({title,description,children}:{title:string;description:string;children?:ReactNode}){return <header className="au-header cm-header"><div><span className="au-eyebrow">WORKSPACE</span><h1>{title}</h1><p>{description}</p></div><div className="au-header-actions">{children}</div></header>;}
export function CommunicationsEmpty({icon,title,description,children}:{icon:ReactNode;title:string;description:string;children?:ReactNode}){return <div className="cm-empty"><span>{icon}</span><h3>{title}</h3><p>{description}</p>{children}</div>;}
